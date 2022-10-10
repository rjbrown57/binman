package binman

import (
	"os"
)

// if user does not provide a -c this will be populated at ~/.config/binman/config
const defaultConfig = `
config:
  releasepath:  #path to keep fetched releases. $HOME/binMan is the default
  tokenvar: #environment variable that contains github token
  upx: #Compress binaries with upx
    enabled: false
    args: [] # arrary of args for upx
releases:
  - repo: rjbrown57/binman
    linkname: # Set link name to be created. Default is to match project name.
    extractfilename: # If the published binary does not match the project name within the tar/zip set the binary name here
    upx: # Upx can also be set per release
      args: [] #["-k","-v"]
    downloadonly: false # binman will only download the file. You take care of the rest ;)
`

// setupConfig will create ~/.config/binman and populate ~/.config/binman/config as needed
func setupConfigDir(configPath string) error {

	err := os.MkdirAll(configPath, 0750)
	if err != nil {
		return err
	}
	return nil
}

// mustEnsureDefaultPaths will create directory and populate config file if necessary
func mustEnsureDefaultPaths() string {

	var binmanConfigPath, binmanConfigFile string

	binmanConfigPath, err := os.UserConfigDir()
	if err != nil {
		log.Fatal("Unable to find config dir")
	}

	binmanConfigPath = binmanConfigPath + "/binman"
	binmanConfigFile = binmanConfigPath + "/config"

	// if the path does not exist we should create it
	if _, err := os.Stat(binmanConfigPath); os.IsNotExist(err) {
		err = setupConfigDir(binmanConfigPath)
		if err != nil {
			log.Fatalf("Unable to create %s", binmanConfigPath)
		}
	}

	// populate the default config if missing
	if _, err := os.Stat(binmanConfigFile); os.IsNotExist(err) {
		// Add the config
		err = writeStringtoFile(binmanConfigFile, defaultConfig)
		if err != nil {
			log.Fatalf("Unable to create %s", binmanConfigFile)
		}
		log.Infof("Populating default config at %s", binmanConfigFile)
	}
	return binmanConfigFile
}
