package cmd

import (
	"bufio"
	"fmt"
	"github.com/ansible-semaphore/semaphore/cli/setup"
	"github.com/ansible-semaphore/semaphore/db"
	"github.com/ansible-semaphore/semaphore/db/factory"
	"github.com/ansible-semaphore/semaphore/util"
	"github.com/spf13/cobra"
	"os"
	"strings"
)

func init() {
	rootCmd.AddCommand(setupCmd)
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Perform interactive setup",
	Run: func(cmd *cobra.Command, args []string) {
		doSetup()
	},
}

//nolint: gocyclo
func doSetup() int {
	var config *util.ConfigType
	for {
		config = &util.ConfigType{}
		config.GenerateSecrets()
		setup.InteractiveSetup(config)

		if setup.AskConfigConfirmation(config) {
			break
		}

		fmt.Println()
	}

	configPath := setup.SaveConfig(config)
	util.Config = config

	fmt.Println(" Pinging db..")

	store := factory.CreateStore()
	if err := store.Connect(); err != nil {
		fmt.Printf("Cannot connect to database!\n %v\n", err.Error())
		os.Exit(1)
	}

	fmt.Println("Running DB Migrations..")
	if err := store.Migrate(); err != nil {
		fmt.Printf("Database migrations failed!\n %v\n", err.Error())
		os.Exit(1)
	}

	stdin := bufio.NewReader(os.Stdin)

	var user db.UserWithPwd
	user.Username = readNewline("\n\n > Username: ", stdin)
	user.Username = strings.ToLower(user.Username)
	user.Email = readNewline(" > Email: ", stdin)
	user.Email = strings.ToLower(user.Email)

	existingUser, err := store.GetUserByLoginOrEmail(user.Username, user.Email)
	util.LogWarning(err)

	if existingUser.ID > 0 {
		// user already exists
		fmt.Printf("\n Welcome back, %v! (a user with this username/email is already set up..)\n\n", existingUser.Name)
	} else {
		user.Name = readNewline(" > Your name: ", stdin)
		user.Pwd = readNewline(" > Password: ", stdin)
		user.Admin = true

		if _, err := store.CreateUser(user); err != nil {
			fmt.Printf(" Inserting user failed. If you already have a user, you can disregard this error.\n %v\n", err.Error())
			os.Exit(1)
		}

		fmt.Printf("\n You are all setup %v!\n", user.Name)
	}

	fmt.Printf(" Re-launch this program pointing to the configuration file\n\n./semaphore --config %v\n\n", configPath)
	fmt.Printf(" To run as daemon:\n\nnohup ./semaphore --config %v &\n\n", configPath)
	fmt.Printf(" You can login with %v or %v.\n", user.Email, user.Username)

	return 0
}

func readNewline(pre string, stdin *bufio.Reader) string {
	fmt.Print(pre)

	str, err := stdin.ReadString('\n')
	util.LogWarning(err)
	str = strings.Replace(strings.Replace(str, "\n", "", -1), "\r", "", -1)

	return str
}
