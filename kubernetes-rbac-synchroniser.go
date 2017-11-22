package main

import (
	"context"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/admin/directory/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	promSuccess = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rbac_synchroniser_success",
			Help: "Cumulative number of role update operations",
		},
		[]string{"count"},
	)

	promErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "rbac_synchroniser_errors",
			Help: "Cumulative number of errors during role update operations",
		},
		[]string{"count"},
	)
)
var address string
var clusterRoleName string
var roleBindingName string
var groupList string
var fakeGroupResponse bool
var kubeConfig string
var inClusterConfig bool
var configFilePath string
var configSubject string
var updateInterval time.Duration

func main() {
	flag.StringVar(&address, "listen-address", ":8080", "The address to listen on for HTTP requests.")
	flag.StringVar(&clusterRoleName, "cluster-role-name", "view", "The cluster role name with permissions.")
	flag.StringVar(&roleBindingName, "rolebinding-name", "developer", "The role binding name per namespace.")
	flag.StringVar(&groupList, "group-list", "", "The group list per namespace comma separated. e.g.: default:group1@test.com,kube-system:group2@test.com")
	flag.BoolVar(&fakeGroupResponse, "fake-group-response", false, "Fake Google Admin API Response. Always response with one group and one member: sync-fake-response@example.com.")
	flag.StringVar(&configFilePath, "config-file-path", "", "The Path to the Service Account's Private Key file. see https://developers.google.com/admin-sdk/directory/v1/guides/delegation")
	flag.StringVar(&configSubject, "config-subject", "", "The Config Subject Email. see https://developers.google.com/admin-sdk/directory/v1/guides/delegation")
	flag.BoolVar(&inClusterConfig, "in-cluster-config", true, "Use in cluster kubeconfig.")
	flag.StringVar(&kubeConfig, "kubeconfig", "", "Absolute path to the kubeconfig file.")
	flag.DurationVar(&updateInterval, "update-interval", 600000, "Update interval in seconds. e.g. 30s or 5m")
	flag.Parse()

	if clusterRoleName == "" {
		log.Println("Missing -cluster-role-name")
		log.Println()
		flag.Usage()
		os.Exit(1)
	}
	if roleBindingName == "" {
		log.Println("Missing -role-name")
		log.Println()
		flag.Usage()
		os.Exit(1)
	}
	if groupList == "" {
		log.Println("Missing -group-list")
		log.Println()
		flag.Usage()
		os.Exit(1)
	}
	if configFilePath == "" {
		log.Println("Missing -config-file-path")
		log.Println()
		flag.Usage()
		os.Exit(1)
	}
	if configSubject == "" {
		log.Println("Missing -config-subject")
		log.Println()
		flag.Usage()
		os.Exit(1)
	}

	stopChan := make(chan struct{}, 1)

	go serveMetrics(address)
	go handleSigterm(stopChan)
	for {
		updateRoles()
		time.Sleep(updateInterval)
	}
}

func handleSigterm(stopChan chan struct{}) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals
	log.Println("Received SIGTERM. Terminating...")
	close(stopChan)
}

// Provides health check and metrics routes
func serveMetrics(address string) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	prometheus.MustRegister(promSuccess)
	prometheus.MustRegister(promErrors)
	http.Handle("/metrics", promhttp.Handler())

	log.Printf("Server listing %v\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

// Gets group users and updates kubernetes rolebindings
func updateRoles() {
	service := getService(configFilePath, configSubject)
	groupListArray := strings.Split(groupList, ",")
	for _, element := range groupListArray {
		elementArray := strings.Split(element, ":")
		namespace, email := elementArray[0], elementArray[1]

		if namespace == "" || email == "" {
			log.Fatalf("Could not update group. Namespace or/and email are empty: %v %v", namespace, email)
			return
		}

		result, error := getMembers(service, email)
		if error != nil {
			log.Fatalf("Unable to get members. %v", error)
			return
		}

		var kubeClusterConfig *rest.Config
		if kubeConfig != "" {
			outOfClusterConfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
			if err != nil {
				log.Fatalf("Unable to get kube config. %v", err)
				return
			}
			kubeClusterConfig = outOfClusterConfig
		} else {
			inClusterConfig, err := rest.InClusterConfig()
			if err != nil {
				log.Fatalf("Unable to get in cluster kube config. %v", err)
			}
			kubeClusterConfig = inClusterConfig
		}
		clientset, err := kubernetes.NewForConfig(kubeClusterConfig)
		if err != nil {
			promErrors.WithLabelValues("get-kube-client").Inc()
			log.Fatalf("Unable to get in kube client. %v", err)
			return
		}

		var subjects []rbacv1beta1.Subject
		for _, member := range result {
			subjects = append(subjects, rbacv1beta1.Subject{
				Kind:     "User",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     member.Email,
			})
		}
		roleBinding := &rbacv1beta1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleBindingName,
				Namespace: namespace,
				Annotations: map[string]string{
					"lastSync": time.Now().UTC().Format(time.RFC3339),
				},
			},
			RoleRef: rbacv1beta1.RoleRef{
				Kind:     "ClusterRole",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     clusterRoleName,
			},
			Subjects: subjects,
		}

		roleClient := clientset.RbacV1beta1().RoleBindings(namespace)
		updateResult, updateError := roleClient.Update(roleBinding)
		if updateError != nil {
			promErrors.WithLabelValues("role-update").Inc()
			log.Fatalf("Unable to update %q rolebinding. %v", roleBindingName, updateError)
			return
		}
		log.Printf("Updated %q rolebinding in %q namespace.\n", updateResult.GetObjectMeta().GetName(), namespace)
		promSuccess.WithLabelValues("role-update").Inc()
	}
}

// Build and returns an Admin SDK Directory service object authorized with
// the service accounts that act on behalf of the given user.
// Args:
//    configFilePath: The Path to the Service Account's Private Key file
//    configSubject: The email of the user. Needs permissions to access the Admin APIs.
// Returns:
//    Admin SDK directory service object.
func getService(configFilePath string, configSubject string) *admin.Service {
	if fakeGroupResponse {
		return nil
	}

	jsonCredentials, err := ioutil.ReadFile(configFilePath)
	if err != nil {
		promErrors.WithLabelValues("get-admin-config").Inc()
		log.Fatalf("Unable to read client secret file: %v", err)
		return nil
	}

	config, err := google.JWTConfigFromJSON(jsonCredentials, admin.AdminDirectoryGroupMemberReadonlyScope, admin.AdminDirectoryGroupReadonlyScope)
	if err != nil {
		promErrors.WithLabelValues("get-admin-config").Inc()
		log.Fatalf("Unable to parse client secret file to config: %v", err)
		return nil
	}
	config.Subject = configSubject
	ctx := context.Background()
	client := config.Client(ctx)

	srv, err := admin.New(client)
	if err != nil {
		promErrors.WithLabelValues("get-admin-client").Inc()
		log.Fatalf("Unable to retrieve Group Settings Client %v", err)
		return nil
	}
	return srv
}

// Gets recursive the group members by email and returns the user list
// Args:
//    service: Admin SDK directory service object.
//    email: The email of the group.
// Returns:
//    Admin SDK member list.
func getMembers(service *admin.Service, email string) ([]*admin.Member, error) {
	if fakeGroupResponse {
		return getFakeMembers(), nil
	}

	result, err := service.Members.List(email).Do()
	if err != nil {
		promErrors.WithLabelValues("get-members").Inc()
		log.Fatalf("Unable to get group members. %v", err)
		return nil, err
	}

	var userList []*admin.Member
	for _, member := range result.Members {
		if member.Type == "GROUP" {
			groupMembers, _ := getMembers(service, member.Email)
			userList = append(userList, groupMembers...)
		} else {
			userList = append(userList, member)
		}
	}

	return userList, nil
}

// Build and returns a fake Admin members object.
// Returns:
//    Admin SDK members object.
func getFakeMembers() []*admin.Member {
	var fakeResult []*admin.Member
	var fakeMember = new(admin.Member)
	fakeMember.Email = "sync-fake-response@example.com"
	fakeResult = append(fakeResult, fakeMember)
	return fakeResult
}
