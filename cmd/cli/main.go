package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path"
	"runtime"
	"time"

	"github.com/pkg/errors"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
)

type app struct {
	filenameWithPath  string
	kubectlUser       string
	kubectlConfigPath string
	kubeloginAlias    string
	kubeloginServer   string
}

type kubeYAML struct {
	APIVersion string `yaml:"apiVersion"`
	Clusters   []struct {
		Cluster map[string]interface{} `yaml:"cluster"`
		Name    string                 `yaml:"name"`
	} `yaml:"clusters"`
	Contexts []struct {
		Context map[string]interface{} `yaml:"context"`
		Name    string                 `yaml:"name"`
	} `yaml:"contexts"`
	CurrentContext string `yaml:"current-context"`
	Kind           string `yaml:"kind"`
	Preferences    struct {
	} `yaml:"preferences"`
	Users []k8User `yaml:"users"`
}

type k8User struct {
	Name string                 `yaml:"name"`
	User map[string]interface{} `yaml:"user"`
}

var (
	aliasName				string
	userFlag				string
	kubeloginServerBaseURL	*url.URL
	doneChannel				chan bool
	kubeloginConfig			string
	kubectlConfig			string
	version  string
)

// Kingpin setup
func loginOrConfigureOptions(cmd *kingpin.CmdClause) {
	switch cmd.FullCommand() {
	case loginCMD.FullCommand():
		cmd.Flag("server-url", "The kubelogin server to connect to. ex: https://kubelogin.example.com").Short('s').URLVar(&kubeloginServerBaseURL)
		cmd.Flag("kubectl-user", "A user in your kubectl config to use for the connection.").Short('u').StringVar(&userFlag)
		cmd.Arg("alias", "An alias to a kubelogin server:kubectl pair.").StringVar(&aliasName)
	case configCMD.FullCommand():
		cmd.Flag("server-url", "The kubelogin server to connect to. ex: https://kubelogin.example.com").Short('s').Required().URLVar(&kubeloginServerBaseURL)
		cmd.Flag("kubectl-user", "A user in your kubectl config to use for the connection.").Short('u').Required().StringVar(&userFlag)
		cmd.Flag("alias", "The alias to use when saving a connection.").Short('a').Required().StringVar(&aliasName)
	}
}

// Kingpin variables sets up sub commands and global arguments.
var (
	// Creates the top level context for all commands flags and arguments
	klogin      		= kingpin.New("kubelogin", "Communicate with a kubelogin server (https://github.com/Nordstrom/kubelogin) and sets the token field of the kubectl config file. The kubernetes API server will use this token for OIDC authentication.")

	// Create some sub commands
	loginCMD			= klogin.Command("login", "Login through a kubelogin server.")
	configCMD	    	= klogin.Command("config", "Create a profile to a kubelogin server.")
	configViewCMD		= klogin.Command("get-config", "View the current kubelogin config.")

	// Global flags
	//TODO: Move this to a different file name. But we need to read the old one if it's there and create the new one so folks don't loose their existing kubelogin settings after an upgrade.

)

// Some trickery which makes things like --server-url required for config, optional for login but not for version or get-config. Based on this https://github.com/alecthomas/kingpin/issues/36
// We are doing this in a function here because we want to use the same variable (I.E. aliasName) with slightly different settings (required for config but not for login.)
// Initialize some kingpin specific settings.
func init() {
	// Set the application version number
	klogin.Version(version)
	// Allow -h as well as --help
	klogin.HelpFlag.Short('h')

	// We have to define these flags separately because they are required for some commands but not others.
	loginCMD.Flag("server-url", "The kubelogin server to connect to. ex: https://kubelogin.example.com").Short('s').URLVar(&kubeloginServerBaseURL)
	loginCMD.Flag("kubectl-user", "A user in your kubectl config to use for the connection.").Short('u').StringVar(&userFlag)
	loginCMD.Arg("alias", "An alias to a kubelogin server:kubectl pair.").StringVar(&aliasName)

	configCMD.Flag("server-url", "The kubelogin server to connect to. ex: https://kubelogin.example.com").Short('s').Required().URLVar(&kubeloginServerBaseURL)
	configCMD.Flag("kubectl-user", "A user in your kubectl config to use for the connection.").Short('u').Required().StringVar(&userFlag)
	configCMD.Flag("alias", "The alias to use when saving a connection.").Short('a').Required().StringVar(&aliasName)

	// Find the home directory of the current user
	user, err := user.Current()
	if err != nil {
		log.Fatalf("Could not determine current user of this system. Err: %v", err)
	}

	klogin.Flag("config-file", "The klogin config file to read from or write to. Default location is $HOME/.kubeloginrc.yaml. Can also be specified by setting the environment variable KUBELOGIN_CONF").Envar("KUBELOGIN_CONF").Default(path.Join(user.HomeDir, "/.kubeloginrc.yaml")).StringVar(&kubeloginConfig)
	klogin.Flag("kubectl-config", "The kubectl config file to write tokens to. Default location is $HOME/.kube/config. Can also be specified by setting the environment variable KUBECTL_CONF").Envar("KUBECTL_CONF").Default(path.Join(user.HomeDir, ".kube", "config")).StringVar(&kubectlConfig)

}

//AliasConfig contains the structure of what's in the config file
type AliasConfig struct {
	Alias       string `yaml:"alias"`
	BaseURL     string `yaml:"server-url"`
	KubectlUser string `yaml:"kubectl-user"`
}

// Config contains the array of aliases (AliasConfig)
type Config struct {
	Aliases []*AliasConfig `yaml:"aliases"`
}

func findFreePort() (string, error) {
	server, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", err
	}
	defer server.Close() // nolint: errcheck
	hostString := server.Addr().String()
	_, portString, err := net.SplitHostPort(hostString)
	if err != nil {
		return "", err
	}
	return portString, nil
}

func (app *app) makeExchange(token string) error {
	url := fmt.Sprintf("%s/exchange?token=%s", app.kubeloginServer, token)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Unable to create request. %s", err)
		return err
	}
	client := http.DefaultClient
	res, err := client.Do(req)
	if err != nil {
		log.Printf("Unable to make request. %s", err)
		return err
	}
	if res.StatusCode != http.StatusOK {
		log.Fatalf("Failed to retrieve token from kubelogin server. Please try again or contact your administrator")
	}
	defer res.Body.Close() // nolint: errcheck
	jwt, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Printf("Unable to read response body. %s", err)
		return err
	}
	if err := app.configureKubectl(string(jwt)); err != nil {
		log.Printf("Error when setting credentials: %v", err)
		return err
	}
	return nil
}

func (app *app) tokenHandler(w http.ResponseWriter, r *http.Request) {
	token := r.FormValue("token")
	if err := app.makeExchange(token); err != nil {
		log.Fatalf("Could not exchange token for jwt %v", err)
	}
	fmt.Fprint(w, "You are now logged in! You can close this window")
	doneChannel <- true
}

// Pure function to test adding/editing token to kubectl config
func editToken(k kubeYAML, a app, t string) kubeYAML {
	found := false
	// Look for existing users which match
	for key, v := range k.Users {
		if a.kubectlUser == v.Name {
			// We only care about a token entry, bypass the issues with client certs
			v.User["token"] = t
			k.Users[key] = v
			found = true
		}
	}
	// We didn't find the user. Time to create one.
	if !found {
		var u k8User
		u.Name = a.kubectlUser
		m := make(map[string]interface{})
		m["token"] = t
		u.User = m
		k.Users = append(k.Users, u)
	}
	return k
}

func (app *app) configureKubectl(jwt string) error {
	// Avoid guessing at appropriate file mode later
	fi, ferr := os.Stat(app.kubectlConfigPath)
	if ferr != nil {
		log.Fatalf("Could not stat kube config: %v", ferr)
	}

	kc, err := ioutil.ReadFile(app.kubectlConfigPath)
	if err != nil {
		log.Fatalf("Could not read kube config file: %v", err)

	}

	var ky kubeYAML
	err = yaml.Unmarshal(kc, &ky)
	if err != nil {
		log.Fatalf("Could not unmarshal kube config: %v", err)

	}

	// Edit or add user in pure function (for testing purposes)
	uy := editToken(ky, *app, jwt)

	out, e := yaml.Marshal(&uy)
	if e != nil {
		log.Fatalf("Could not write kube config: %v", e)

	}
	return ioutil.WriteFile(app.kubectlConfigPath, out, fi.Mode())
}

func (app *app) generateAuthURL() (string, string, error) {
	portNum, err := findFreePort()
	if err != nil {
		log.Print("err, could not find an open port")
		return "", "", err
	}

	loginURL := fmt.Sprintf("%s/login?port=%s", app.kubeloginServer, portNum)

	return loginURL, portNum, nil
}

func createMux(app app) *http.ServeMux {
	newMux := http.NewServeMux()
	newMux.HandleFunc("/exchange/", app.tokenHandler)
	newMux.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(""))
		if err != nil {
			log.Printf("Unable to write favicon: %v ", err)
		}
		return
	})
	return newMux
}

func generateURLAndListenForServerResponse(app app) {
	loginURL, portNum, err := app.generateAuthURL()
	if err != nil {
		log.Fatal(err.Error())
	}
	doneChannel = make(chan bool)
	go func() {
		l, err := net.Listen("tcp", ":"+portNum)
		if err != nil {
			fmt.Printf("Error listening on port: %s. Error: %v\n", portNum, err)
			os.Exit(1)
		}
		if runtime.GOOS == "darwin" {
			// On OS X, run the `open` CLI to use the default browser to open the login URL.
			fmt.Printf("Opening %s ...\n", loginURL)
			err := exec.Command("/usr/bin/open", loginURL).Run()
			if err != nil {
				fmt.Printf("Error opening; please open the URL manually: %s \n", loginURL)
			}
		} else {
			fmt.Printf("Follow this URL to log into auth provider: %s\n", loginURL)
		}
		if err = http.Serve(l, createMux(app)); err != nil {
			fmt.Printf("Error listening on port: %s. Error: %v\n", portNum, err)
			os.Exit(1)
		}
	}()
	<-doneChannel
	fmt.Println("You are now logged in! Enjoy kubectl-ing!")
	time.Sleep(1 * time.Second)
}

func (app *app) getConfigSettings(alias string) error {
	yamlFile, err := ioutil.ReadFile(app.filenameWithPath)
	if err != nil {
		return errors.Wrap(err, "failed to read config file for login use")
	}
	var config Config
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return errors.Wrap(err, "failed to unmarshal yaml file for login use")
	}

	aliasConfig, ok := config.aliasSearch(alias)
	if !ok {
		return errors.New("Could not find specified alias, check spelling or use the config verb to create an alias")
	}
	app.kubectlUser = aliasConfig.KubectlUser
	app.kubeloginServer = aliasConfig.BaseURL
	return nil
}

func (config *Config) aliasSearch(alias string) (*AliasConfig, bool) {
	for index, aliases := range config.Aliases {
		if alias == aliases.Alias {
			return config.Aliases[index], true
		}
	}
	return nil, false
}

func (config *Config) createConfig(onDiskFile string, aliasConfig AliasConfig) error {
	log.Print("Couldn't find config file in root directory. Creating config file...")
	_, e := os.Stat(onDiskFile) // Does config file exist?
	if os.IsNotExist(e) {       // Create file
		fh, err := os.Create(onDiskFile)
		if err != nil {
			return errors.Wrap(err, "failed to create file in root directory")
		}
		_ = fh.Close()
	}

	log.Print("Config file created, setting config values...")
	config.Aliases = make([]*AliasConfig, 0)
	config.appendAlias(aliasConfig)
	if err := config.writeToFile(onDiskFile); err != nil {
		log.Fatal(err)
	}
	log.Print("File configured")
	return nil
}

func (config *Config) newAliasConfig(kubeloginrcAlias, loginServerURL, kubectlUser string) AliasConfig {
	newConfig := AliasConfig{
		BaseURL:     loginServerURL,
		Alias:       kubeloginrcAlias,
		KubectlUser: kubectlUser,
	}
	return newConfig
}

func (config *Config) appendAlias(aliasConfig AliasConfig) {
	config.Aliases = append(config.Aliases, &aliasConfig)
}

func (config *Config) writeToFile(onDiskFile string) error {
	marshaledYaml, err := yaml.Marshal(config)
	if err != nil {
		return errors.Wrap(err, "failed to marshal alias yaml")
	}
	if err := ioutil.WriteFile(onDiskFile, marshaledYaml, 0600); err != nil {
		return errors.Wrap(err, "failed to write to kubeloginrc file with the alias")
	}
	log.Printf(string(marshaledYaml))
	return nil
}

func (config *Config) updateAlias(aliasConfig *AliasConfig, loginServerURL *url.URL, onDiskFile string) error {
	aliasConfig.KubectlUser = userFlag
	aliasConfig.BaseURL = loginServerURL.String()
	if err := config.writeToFile(onDiskFile); err != nil {
		log.Fatal(err)
	}
	log.Print("Alias updated")
	return nil
}

func (app *app) configureFile(kubeloginrcAlias string, loginServerURL *url.URL, kubectlUser string) error {
	var config Config
	aliasConfig := config.newAliasConfig(kubeloginrcAlias, loginServerURL.String(), kubectlUser)
	yamlFile, err := ioutil.ReadFile(app.filenameWithPath)
	if err != nil {
		return config.createConfig(app.filenameWithPath, aliasConfig) // Either error or nil value
	}
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return errors.Wrap(err, "Failed to parse kubectl config file.")
	}
	foundAliasConfig, ok := config.aliasSearch(aliasName)
	if !ok {
		newConfig := config.newAliasConfig(kubeloginrcAlias, loginServerURL.String(), kubectlUser)
		config.appendAlias(newConfig)
		if err := config.writeToFile(app.filenameWithPath); err != nil {
			log.Fatal(err)
		}
		log.Print("New Alias configured")
		return nil
	}

	return config.updateAlias(foundAliasConfig, loginServerURL, app.filenameWithPath) // Either error or nil value
}

// Print version number and exit.
func printKubeloginVersion() {
	log.Fatalln("Version number currently does not exist")
}

// Prints the current kubelogin client config and exits.
func (app *app) printKubeloginConfig() error {
	yamlFile, err := ioutil.ReadFile(app.filenameWithPath)
	if err != nil {
		return errors.Wrap(err, "Failed to read kubelogin config file.")
	}
	var config Config
	if err := yaml.Unmarshal(yamlFile, &config); err != nil {
		return errors.Wrap(err, "Failed to parse kubelogin config file.")
	}

	fmt.Print(string(yamlFile))

	return nil
}

func main() {
	var app app

	app.filenameWithPath = kubeloginConfig

	app.kubectlConfigPath = kubectlConfig

	// Parse kingpin args
	thisCommand := kingpin.MustParse(klogin.Parse(os.Args[1:]))

	switch thisCommand {
	case loginCMD.FullCommand():

		// I wish there was a cleaner way to say that you can provide an alias OR both --kubectl-user and --server-url
		if aliasName == "" && kubeloginServerBaseURL == nil {
			klogin.FatalUsage("Either --server-url or a configured alias must be provided")
		}

		// If the have provided a URL
		if kubeloginServerBaseURL != nil {
			// Ensure that they provided a user as well.
			if userFlag == "" {
				klogin.FatalUsage("--kubectl-user must be provided when using --server-url with login")
			}

			// If so use both of those values.
			app.kubectlUser = userFlag
			app.kubeloginServer = kubeloginServerBaseURL.String()
		}

		// Otherwise they have provided an alias so we will fetch user and URL from the config.
		err := app.getConfigSettings(aliasName)

		if err != nil {
			log.Fatal(err)
		}

		generateURLAndListenForServerResponse(app)
	case configCMD.FullCommand():
		// We no longer need to verify the URL since kingpin will be doing it for us.
		//verifiedServerURL, err := url.ParseRequestURI(kubeloginServerBaseURL)
		if err := app.configureFile(aliasName, kubeloginServerBaseURL, userFlag); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	case configViewCMD.FullCommand():
		app.printKubeloginConfig()
	}
}
