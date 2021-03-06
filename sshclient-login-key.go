// Interactive SSH login with SSH key.

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"

	// Importing crypto/ssh
	"golang.org/x/crypto/ssh"
)

var (
	username, serverIP, serverPort, pKeyFile string
)

// Read flags
func init() {
	flag.StringVar(&serverPort, "port", "22", "SSH server port")
	flag.StringVar(&serverIP, "ip", "127.0.0.1", "SSH server IP")
	flag.StringVar(&username, "user", "", "username")
	flag.StringVar(&pKeyFile, "pkey", "", "unencrypted private key file")
}

func main() {
	// Parse flags
	flag.Parse()

	// Check if username and pKey have been submitted
	if username == "" {
		fmt.Println("Must supply username")
		os.Exit(2)
	}

	if pKeyFile == "" {
		fmt.Println("Must supply private key")
		os.Exit(2)
	}

	// Now we must read the private key
	pKey, err := ioutil.ReadFile(pKeyFile)
	if err != nil {
		fmt.Println("Failed to read private key from file", err)
		os.Exit(2)
	}

	// Create a signer with the private key
	signer, err := ssh.ParsePrivateKey(pKey)
	if err != nil {
		fmt.Println("Failed to parse private key", err)
		os.Exit(2)
	}

	// Create SSH config
	config := &ssh.ClientConfig{
		// Username
		User: username,
		// Each config must have one AuthMethod. Now we use key
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		// This callback function validates the server.
		// Danger! We are ignoring host info
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	// Server address
	t := net.JoinHostPort(serverIP, serverPort)

	// Connect to the SSH server
	sshConn, err := ssh.Dial("tcp", t, config)
	if err != nil {
		fmt.Printf("Failed to connect to %v\n", t)
		fmt.Println(err)
		os.Exit(2)
	}

	// Create new SSH session
	session, err := sshConn.NewSession()
	if err != nil {
		fmt.Printf("Cannot create SSH session to %v\n", t)
		fmt.Println(err)
		os.Exit(2)
	}

	// Close the session when main returns
	defer session.Close()

	// For an interactive session we must redirect IO
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	input, err := session.StdinPipe()
	if err != nil {
		fmt.Println("Error redirecting session input", err)
		os.Exit(2)
	}

	// Setup terminal mode when requesting pty. You can see all terminal modes at
	// https://github.com/golang/crypto/blob/master/ssh/session.go#L56 or read
	// the RFC for explanation https://tools.ietf.org/html/rfc4254#section-8
	termModes := ssh.TerminalModes{
		ssh.ECHO: 0, // Disable echo
	}

	// Request pty
	// https://tools.ietf.org/html/rfc4254#section-6.2
	// First variable is term environment variable value which specifies terminal.
	// term doesn't really matter here, we will use "vt220".
	// Next are height and width: (40,80) characters and finall termModes.
	err = session.RequestPty("vt220", 40, 80, termModes)
	if err != nil {
		fmt.Println("RequestPty failed", err)
		os.Exit(2)
	}

	// Also
	// if err = session.RequestPty("vt220", 40, 80, termModes); err != nil {
	//  fmt.Println("RequestPty failed", err)
	//  os.Exit(2)
	// }

	// Now we can start a remote shell
	err = session.Shell()
	if err != nil {
		fmt.Println("shell failed", err)
		os.Exit(2)
	}

	// Same as above, a different way to check for errors
	// if err = session.Shell(); err != nil {
	//  fmt.Println("shell failed", err)
	//  os.Exit(2)
	// }

	// Endless loop to capture commands
	// Note: After exit, we need to ctrl+c to end the application.
	for {
		io.Copy(input, os.Stdin)
	}
}
