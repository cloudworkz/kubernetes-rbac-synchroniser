package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/admin/directory/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	roleUpdates = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "role_updates",
			Help: "Cumulative number of role update operations",
		},
		[]string{"count"},
	)

	roleUpdateErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "role_update_errors",
			Help: "Cumulative number of errors during role update operations",
		},
		[]string{"count"},
	)
)
var address string
var clusterRoleName string
var roleName string
var groupList string
var fakeGroupResponse bool
var kubeConfig string
var inClusterConfig bool
var token string
var tokenFilePath string

func main() {
	flag.StringVar(&address, "listen-address", ":8080", "The address to listen on for HTTP requests.")
	flag.StringVar(&clusterRoleName, "cluster-role-name", "developer", "The cluster role name with permissions.")
	flag.StringVar(&roleName, "role-name", "developer", "The role binding name per namespace.")
	flag.StringVar(&groupList, "group-list", "default:group1@test.com,kube-system:group2@test.com", "The group list per namespace comma separated.")
	flag.BoolVar(&fakeGroupResponse, "fake-group-response", false, "Fake Google Admin API Response.")
	flag.StringVar(&token, "token", "", "The google group setting API token.")
	flag.StringVar(&tokenFilePath, "token-file-path", "", "The file with google group setting file.")
	flag.BoolVar(&inClusterConfig, "in-cluster-config", true, "Use in cluster kubeconfig.")
	flag.StringVar(&kubeConfig, "kubeconfig", "", "Absolute path to the kubeconfig file.")
	flag.Parse()

	if clusterRoleName == "" {
		log.Println("Missing cluster-role-name")
		log.Println()
		flag.Usage()
		os.Exit(1)
	}
	if roleName == "" {
		log.Println("Missing role-name")
		log.Println()
		flag.Usage()
		os.Exit(1)
	}
	if groupList == "" {
		log.Println("Missing groupList")
		log.Println()
		flag.Usage()
		os.Exit(1)
	}

	stopChan := make(chan struct{}, 1)

	go serveMetrics(address)
	go handleSigterm(stopChan)
	for {
		go updateRoles()
		time.Sleep(time.Second * 30)
	}
}

func handleSigterm(stopChan chan struct{}) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM)
	<-signals
	log.Println("Received SIGTERM. Terminating...")
	close(stopChan)
}

func serveMetrics(address string) {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	prometheus.MustRegister(roleUpdates)
	prometheus.MustRegister(roleUpdateErrors)
	http.Handle("/metrics", promhttp.Handler())

	log.Printf("Server listing %v\n", address)
	log.Fatal(http.ListenAndServe(address, nil))
}

func updateRoles() {
	var service *admin.Service
	if !fakeGroupResponse {
		ctx := context.Background()

		b, err := ioutil.ReadFile(filepath.Join(".credentials", "client_secret.json"))
		if err != nil {
			roleUpdateErrors.WithLabelValues("get-admin-config").Inc()
			log.Fatalf("Unable to read client secret file: %v", err)
		}

		config, err := google.ConfigFromJSON(b, admin.AdminDirectoryGroupMemberReadonlyScope)
		if err != nil {
			roleUpdateErrors.WithLabelValues("get-admin-config").Inc()
			log.Fatalf("Unable to parse client secret file to config: %v", err)
		}
		client := getClient(ctx, config)

		srv, err := admin.New(client)
		if err != nil {
			roleUpdateErrors.WithLabelValues("get-admin-client").Inc()
			log.Fatalf("Unable to retrieve Group Settings Client %v", err)
		}
		service = srv
	}
	groupListArray := strings.Split(groupList, ",")
	for _, element := range groupListArray {
		elementArray := strings.Split(element, ":")
		namespace, email := elementArray[0], elementArray[1]

		if namespace == "" || email == "" {
			log.Fatalf("Could not update group. Namespace or/and email are empty: %v %v", namespace, email)
		}
		//log.Printf("email %q.\n", email)
		//log.Printf("srv %q.\n", srv)

		var result *admin.Members
		if fakeGroupResponse {
			var fakeResult = new(admin.Members)
			var fakeMember = new(admin.Member)
			fakeMember.Email = "sync-fake@test.com"
			fakeResult.Members = append(fakeResult.Members, fakeMember)
			result = fakeResult
		} else {
			apiResult, err := service.Members.List(email).Do()
			if err != nil {
				roleUpdateErrors.WithLabelValues("get-members").Inc()
				log.Fatalf("Unable to retrieve group settings. %v", err)
			}
			result = apiResult
		}

		// log.Printf("GROUPS %q.\n", result)
		var kubeClusterConfig *rest.Config
		if kubeConfig != "" {
			outofclusterconfig, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
			if err != nil {
				log.Fatalf("Unable to get kube config. %v", err)
			}
			kubeClusterConfig = outofclusterconfig
		} else {
			inclusterkubeconfig, err := rest.InClusterConfig()
			if err != nil {
				log.Fatalf("Unable to get in cluster kube config. %v", err)
			}
			kubeClusterConfig = inclusterkubeconfig
		}
		clientset, err := kubernetes.NewForConfig(kubeClusterConfig)
		if err != nil {
			roleUpdateErrors.WithLabelValues("get-kube-client").Inc()
			log.Fatalf("Unable to get in kube client. %v", err)
		}
		var subjects []rbacv1beta1.Subject
		for _, member := range result.Members {
			subjects = append(subjects, rbacv1beta1.Subject{
				Kind:     "User",
				APIGroup: "rbac.authorization.k8s.io",
				Name:     member.Email,
			})
		}
		roleBinding := &rbacv1beta1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      roleName,
				Namespace: namespace,
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
			roleUpdateErrors.WithLabelValues("role-update").Inc()
			log.Fatalf("Unable to update role. %v", updateError)
		}
		log.Printf("Updated %q.\n", updateResult.GetObjectMeta().GetName())
		roleUpdates.WithLabelValues("role-update").Inc()
	}
}

func getClient(ctx context.Context, config *oauth2.Config) *http.Client {
	if tokenFilePath == "" {
		cacheFile := tokenCacheFile()
		if cacheFile == "" {
			log.Fatalf("Unable to get path to cached credential file.")
		}
		tokenFilePath = cacheFile
	}

	tok, err := tokenFromFile(tokenFilePath)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokenFilePath, tok)
	}
	return config.Client(ctx, tok)
}

func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	if token == "" {
		authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
		log.Fatalf("Unable to generate oauth2 token. Go to the following link in your browser "+
			"then use '-token' flag for the authorization code: \n%v\n", authURL)
	}

	tok, err := config.Exchange(oauth2.NoContext, token)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web %v", err)
	}
	return tok
}

func tokenCacheFile() string {
	if tokenFilePath == "" {
		tokenFilePath = filepath.Join(".", ".credentials")
	}
	os.MkdirAll(tokenFilePath, 0700)
	return filepath.Join(tokenFilePath,
		url.QueryEscape("kubernetes-rbac-synchroniser.json"))
}

func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	t := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(t)
	defer f.Close()
	return t, err
}

func saveToken(file string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", file)
	f, err := os.OpenFile(file, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		log.Fatalf("Unable to cache oauth token: %v", err)
	}
	defer f.Close()
	json.NewEncoder(f).Encode(token)
}
