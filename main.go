// +build go1.12,linux,amd64

//go:generate jade -writer -basedir static/templates -d ./static/jade welcomeannounce/index.jade home/index.jade login/index.jade about/index.jade entity/index.jade entity/create.jade product/index.jade product/create.jade storage/index.jade storage/create.jade storelocation/index.jade storelocation/create.jade person/index.jade person/create.jade person/pupdate.jade search.jade menu.jade
//go:generate go run . -genlocalejs
//go:generate rice embed-go
package main

// build with
//go build -trimpath -ldflags "-X globals.BuildID=$(git tag | head -1)" -o gochimitheque

import (
	"database/sql"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	rice "github.com/GeertJohan/go.rice"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/tbellembois/gochimitheque/datastores"
	"github.com/tbellembois/gochimitheque/handlers"
	"github.com/tbellembois/gochimitheque/logger"
	"github.com/tbellembois/gochimitheque/mailer"
	"github.com/tbellembois/gochimitheque/models"
	"github.com/tbellembois/gochimitheque/static/localejs"
)

var (
	env handlers.Env

	// Starting parameters and commands.
	paramListenPort,
	paramDBPath,
	paramAdminList,
	paramLogFile,
	commandImportV1From,
	commandImportFrom,
	commandMailTest *string
	paramPublicProductsEndpoint,
	commandResetAdminPassword,
	commandUpdateQRCode,
	paramDebug,
	commandVersion,
	commandGenLocaleJS,
	paramDisableCache *bool
)

// TimeTrack displays the run time of the function "name"
// from the start time "start"
// use: defer utils.TimeTrack(time.Now(), "GetProducts")
// at the begining of the function to track
// func TimeTrack(start time.Time, name string) {
// 	elapsed := time.Since(start)
// 	logger.Log.Debug(fmt.Sprintf("%s took %s", name, elapsed))
// }

func init() {

	env = handlers.NewEnv()

	// Configuration parameters.
	flagListenPort := flag.String("listenport", "8081", "the port to listen")
	flagDBPath := flag.String("dbpath", "./", "the application sqlite directory path")
	flagProxyURL := flag.String("proxyurl", "", "the application url (without the path) if behind a proxy, with NO trailing /")
	flagProxyPath := flag.String("proxypath", "/", "the application path if behind a proxy, with the trailing /")
	flagMailServerAddress := flag.String("mailserveraddress", "localhost", "the mail server address")
	flagMailServerPort := flag.String("mailserverport", "25", "the mail server port")
	flagMailServerSender := flag.String("mailserversender", "", "the mail server sender")
	flagMailServerUseTLS := flag.Bool("mailserverusetls", false, "use TLS? (optional)")
	flagMailServerTLSSkipVerify := flag.Bool("mailservertlsskipverify", false, "skip TLS verification? (optional)")
	flagPublicProductsEndpoint := flag.Bool("enablepublicproductsendpoint", false, "enable public products endpoint (optional)")
	flagAdminList := flag.String("admins", "", "the additional admins (comma separated email adresses) (optional) ")
	flagLogFile := flag.String("logfile", "", "log to the given file (optional)")
	flagDebug := flag.Bool("debug", false, "debug (verbose log), default is error")
	flagDisableCache := flag.Bool("disablecache", false, "disable the cache (development only)")

	// One shot commands.
	flagResetAdminPassword := flag.Bool("resetadminpassword", false, "reset the admin password to `chimitheque`")
	flagUpdateQRCode := flag.Bool("updateqrcode", false, "regenerate storages QR codes")
	flagVersion := flag.Bool("version", false, "display application version")
	flagMailTest := flag.String("mailtest", "", "send a test mail")
	flagImportV1From := flag.String("importv1from", "", "full path of the directory containing the Chimithèque v1 CSV to import")
	flagImportFrom := flag.String("importfrom", "", "base URL of the external Chimithèque instance (running with -enablepublicproductsendpoint) to import products from")
	flagGenLocaleJS := flag.Bool("genlocalejs", false, "generate JS locales (developper target)")

	flag.Parse()

	paramListenPort = flagListenPort
	paramDBPath = flagDBPath
	env.ProxyURL = *flagProxyURL
	env.ProxyPath = *flagProxyPath
	mailer.MailServerAddress = *flagMailServerAddress
	mailer.MailServerPort = *flagMailServerPort
	mailer.MailServerSender = *flagMailServerSender
	mailer.MailServerUseTLS = *flagMailServerUseTLS
	mailer.MailServerTLSSkipVerify = *flagMailServerTLSSkipVerify
	paramPublicProductsEndpoint = flagPublicProductsEndpoint
	paramAdminList = flagAdminList
	paramLogFile = flagLogFile
	paramDebug = flagDebug
	paramDisableCache = flagDisableCache

	commandResetAdminPassword = flagResetAdminPassword
	commandUpdateQRCode = flagUpdateQRCode
	commandVersion = flagVersion
	commandMailTest = flagMailTest
	commandImportV1From = flagImportV1From
	commandImportFrom = flagImportFrom
	commandGenLocaleJS = flagGenLocaleJS

	if env.ProxyURL != "" {
		env.ApplicationFullURL = env.ProxyURL + env.ProxyURL
	} else {
		env.ApplicationFullURL = "http://localhost:" + *paramListenPort
	}

	if env.BuildID == "" {
		env.BuildID = time.Now().Format("2006-02-01")
	}

}

func initLogger() {

	var err error

	if *paramDebug {
		logger.Log.SetLevel(logrus.DebugLevel)
	} else {
		logger.Log.SetLevel(logrus.InfoLevel)
	}

	if *paramLogFile != "" {

		var commandLineLogFile *os.File
		if commandLineLogFile, err = os.OpenFile(*paramLogFile, os.O_WRONLY|os.O_CREATE, 0755); err != nil {
			logger.Log.Fatal(err)
		} else {
			logger.Log.SetOutput(commandLineLogFile)
		}
		defer commandLineLogFile.Close()

	}

	var internalServerErrorLogFile *os.File
	if internalServerErrorLogFile, err = os.OpenFile("errors.log", os.O_WRONLY|os.O_CREATE, 0755); err != nil {
		logger.Log.Fatal(err)
	} else {
		logger.LogInternal.SetOutput(internalServerErrorLogFile)
		logger.LogInternal.SetReportCaller(true)
	}
	defer internalServerErrorLogFile.Close()

}

func initDB() {

	var (
		err       error
		datastore datastores.Datastore
	)

	dbname := path.Join(*paramDBPath, "storage.db")
	logger.Log.Info("- opening database connection to " + dbname)
	if datastore, err = datastores.NewSQLiteDBstore(dbname); err != nil {
		logger.Log.Fatal(err)
	}

	logger.Log.Info("- creating database if needed")
	if err = datastore.CreateDatabase(); err != nil {
		logger.Log.Fatal(err)
	}

	logger.Log.Info("- running maintenance job")
	datastore.Maintenance()

	env.DB = datastore

}

func initAdmins() {

	var (
		err           error
		p             models.Person
		formerAdmins  []models.Person
		currentAdmins []string
		isStillAdmin  bool
	)

	if *paramAdminList != "" {
		currentAdmins = strings.Split(*paramAdminList, ",")
	}

	if formerAdmins, err = env.DB.GetAdmins(); err != nil {
		logger.Log.Fatal(err)
	}

	// Cleaning former admins.
	for _, fa := range formerAdmins {

		isStillAdmin = false
		logger.Log.Info("former admin: " + fa.PersonEmail)
		for _, ca := range currentAdmins {
			if ca == fa.PersonEmail {
				isStillAdmin = true
			}
		}
		if !isStillAdmin {
			logger.Log.Info(fa.PersonEmail + " is not an admin anymore, removing permissions")
			if err = env.DB.UnsetPersonAdmin(fa.PersonID); err != nil {
				logger.Log.Fatal(err)
			}
		}

	}
	// Setting up new ones.
	if len(currentAdmins) > 0 {

		for _, ca := range currentAdmins {
			logger.Log.Info("additional admin: " + ca)
			if p, err = env.DB.GetPersonByEmail(ca); err != nil {
				if err == sql.ErrNoRows {
					logger.Log.Fatal("user " + ca + " not found in database")
				} else {
					logger.Log.Fatal(err)
				}
			}

			if err = env.DB.SetPersonAdmin(p.PersonID); err != nil {
				logger.Log.Fatal(err)
			}
		}

	}

}

func initStaticResources(router *mux.Router) {

	casbinModelBox := rice.MustFindBox("models")
	modelf, e := casbinModelBox.Open("model.conf")
	if e != nil {
		logger.Log.Error("model.conf load from box error: " + e.Error())
		os.Exit(1)
	}
	models, e := modelf.Stat()
	if e != nil {
		logger.Log.Error("model.conf stat error: " + e.Error())
		os.Exit(1)
	}

	modelb := make([]byte, models.Size()-1)
	_, e = modelf.Read(modelb)
	if e != nil {
		logger.Log.Error("model.conf load error: " + e.Error())
		os.Exit(1)
	}

	env.CasbinModel = string(modelb)

	webfontsBox := rice.MustFindBox("static/webfonts")
	webfontsFileServer := http.StripPrefix("/webfonts/", http.FileServer(webfontsBox.HTTPBox()))
	http.Handle("/webfonts/", webfontsFileServer)

	fontsBox := rice.MustFindBox("static/fonts")
	fontsFileServer := http.StripPrefix("/fonts/", http.FileServer(fontsBox.HTTPBox()))
	http.Handle("/fonts/", fontsFileServer)

	cssBox := rice.MustFindBox("static/css")
	cssFileServer := http.StripPrefix("/css/", http.FileServer(cssBox.HTTPBox()))
	http.Handle("/css/", cssFileServer)

	jsBox := rice.MustFindBox("static/js")
	jsFileServer := http.StripPrefix("/js/", http.FileServer(jsBox.HTTPBox()))
	http.Handle("/js/", jsFileServer)

	imgBox := rice.MustFindBox("static/img")
	imgFileServer := http.StripPrefix("/img/", http.FileServer(imgBox.HTTPBox()))
	http.Handle("/img/", imgFileServer)

	wasmBox := rice.MustFindBox("wasm")
	wasmFileServer := http.StripPrefix("/wasm/", http.FileServer(wasmBox.HTTPBox()))
	http.Handle("/wasm/", wasmFileServer)

	http.Handle("/", router)

}

func main() {

	var (
		err error
	)

	// Basic commands.
	if *commandVersion {
		fmt.Println(env.BuildID)
		os.Exit(0)
	}

	if *commandGenLocaleJS {
		localejs.GenerateLocalJS()
		os.Exit(0)
	}

	initLogger()

	logger.Log.Debugf("- env: %+v", env)
	logger.Log.Info("- application version: " + env.BuildID)
	logger.Log.Info("- application endpoint: " + env.ApplicationFullURL)

	initDB()

	// Advanced commands.
	if *commandImportV1From != "" {

		logger.Log.Info("- import from Chimithèque v1 csv into database")
		err := env.DB.ImportV1(*commandImportV1From)
		if err != nil {
			logger.Log.Error("an error occured: " + err.Error())
			os.Exit(1)
		}
		os.Exit(0)

	}

	if *commandImportFrom != "" {

		logger.Log.Info("- import from URL into database")
		err := env.DB.Import(*commandImportFrom)
		if err != nil {
			logger.Log.Error("an error occured: " + err.Error())
			os.Exit(1)
		}
		os.Exit(0)

	}

	if *commandResetAdminPassword {

		logger.Log.Info("- reseting admin password to `chimitheque`")
		a, err := env.DB.GetPersonByEmail("admin@chimitheque.fr")
		if err != nil {
			logger.Log.Error("an error occured: " + err.Error())
			os.Exit(1)
		}
		a.PersonPassword = "chimitheque"
		err = env.DB.UpdatePersonPassword(a)
		if err != nil {
			logger.Log.Error("an error occured: " + err.Error())
			os.Exit(1)
		}
		os.Exit(0)

	}

	if *commandUpdateQRCode {

		logger.Log.Info("- updating storages QR codes")
		err := env.DB.UpdateAllQRCodes()
		if err != nil {
			logger.Log.Error("an error occured: " + err.Error())
			os.Exit(1)
		}
		os.Exit(0)

	}

	if *commandMailTest != "" {

		logger.Log.Info("- sending a mail to " + *commandMailTest)
		err := mailer.TestMail(*commandMailTest)
		if err != nil {
			logger.Log.Error("an error occured: " + err.Error())
			os.Exit(1)
		}
		os.Exit(0)

	}

	initAdmins()

	router := buildEndpoints()

	initStaticResources(router)

	env.InitCasbinPolicy()

	logger.Log.Info("- application running")
	if err = http.ListenAndServe(":"+*paramListenPort, nil); err != nil {
		panic("error running the server")
	}

}
