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
	"google.golang.org/api/groupssettings/v1"
	rbacv1beta1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
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
var kubeconfig *string
var token string
var tokenFilePath string

func main() {
	flag.StringVar(&address, "listen-address", ":8080", "The address to listen on for HTTP requests.")
	flag.StringVar(&clusterRoleName, "cluster-role-name", "developer", "The cluster role name with permissions.")
	flag.StringVar(&roleName, "role-name", "developer", "The role binding name per namespace.")
	flag.StringVar(&groupList, "group-list", "default:group1@test.com,kube-system:group2@test.com", "The group list per namespace comma separated.")
	flag.StringVar(&token, "token", "", "The google group setting API token.")
	flag.StringVar(&tokenFilePath, "token-file-path", "", "The file with google group setting file.")
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
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
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err)
	}

	stopChan := make(chan struct{}, 1)

	go serveMetrics(address)
	go handleSigterm(stopChan)
	for {
		go updateRoles(config)
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

func updateRoles(kubeconfig *rest.Config) {
	ctx := context.Background()

	b, err := ioutil.ReadFile(filepath.Join(".credentials", "client_secret.json"))
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	config, err := google.ConfigFromJSON(b, admin.AdminDirectoryGroupReadonlyScope, groupssettings.AppsGroupsSettingsScope)
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(ctx, config)

	srv, err := admin.New(client)
	if err != nil {
		log.Fatalf("Unable to retrieve Group Settings Client %v", err)
	}
	groupListArray := strings.Split(groupList, ",")
	for _, element := range groupListArray {
		elementArray := strings.Split(element, ":")
		namespace, email := elementArray[0], elementArray[1]

		if namespace == "" || email == "" {
			log.Fatalf("Could not update group. Namespace or/and email are empty: %v %v", namespace, email)
		}
		log.Printf("email %q.\n", email)
		r, err := srv.Groups.Get(email).Do()
		if err != nil {
			log.Fatalf("Unable to retrieve group settings. %v", err)
		}

		log.Printf("GROUPS %q.\n", r)
		log.Printf("%s - %s", r.Email, r.Description)
		clientset, err := kubernetes.NewForConfig(kubeconfig)
		if err != nil {
			panic(err)
		}

		roleBinding := &rbacv1beta1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: roleName,
			},
		}

		roleClient := clientset.RbacV1beta1().RoleBindings(namespace)
		result, err := roleClient.Update(roleBinding)
		if err != nil {
			panic(err)
		}
		log.Printf("Updated %q.\n", result.GetObjectMeta().GetName())
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
