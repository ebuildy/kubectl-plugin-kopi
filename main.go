package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

func main() {
	flags := genericclioptions.NewConfigFlags(true)

	pflag.CommandLine = pflag.NewFlagSet("kopi", pflag.ExitOnError)
	flags.AddFlags(pflag.CommandLine)
	pflag.Parse()

	args := pflag.Args()
	if len(args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: kopi [-n namespace] <secret|cm> <name> <key>")
		fmt.Fprintln(os.Stderr, "  secret|cm: secret or configmap")
		fmt.Fprintln(os.Stderr, "  name: name of the secret or configmap")
		fmt.Fprintln(os.Stderr, "  key: key to copy from the secret or configmap")
		os.Exit(1)
	}

	loader := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{
			AuthInfo:    clientcmdapi.AuthInfo{},
			ClusterInfo: clientcmdapi.Cluster{},
		},
	)

	ns, _, err := loader.Namespace()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting namespace: %v\n", err)
		os.Exit(1)
	}
	if ns == "" {
		ns = "default"
	}

	resourceType := args[0]
	name := args[1]
	key := args[2]

	config, err := loader.ClientConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load kubeconfig: %v\n", err)
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create clientset: %v\n", err)
		os.Exit(1)
	}

	ctx := context.Background()
	var value string

	switch resourceType {
	case "secret":
		secret, err := clientset.CoreV1().Secrets(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get secret %s: %v\n", name, err)
			os.Exit(1)
		}
		data, ok := secret.Data[key]
		if !ok {
			fmt.Fprintf(os.Stderr, "Key %s not found in secret %s\n", key, name)
			os.Exit(1)
		}
		value = string(data)
	case "cm", "configmap":
		cm, err := clientset.CoreV1().ConfigMaps(ns).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to get configmap %s: %v\n", name, err)
			os.Exit(1)
		}
		data, ok := cm.Data[key]
		if !ok {
			fmt.Fprintf(os.Stderr, "Key %s not found in configmap %s\n", key, name)
			os.Exit(1)
		}
		value = data
	default:
		fmt.Fprintf(os.Stderr, "Unknown resource type: %s (use 'secret' or 'cm')\n", resourceType)
		os.Exit(1)
	}

	if err := copyToClipboard(value); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to copy to clipboard: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Copied %s/%s.%s to clipboard\n", ns, name, key)
}

func copyToClipboard(value string) error {
	var cmd *exec.Cmd

	commands := []string{"wl-copy", "xclip", "xsel", "pbcopy", "clip.exe"}

	for _, c := range commands {
		if _, err := exec.LookPath(c); err == nil {
			switch c {
			case "wl-copy":
				cmd = exec.Command("wl-copy")
			case "xclip":
				cmd = exec.Command("xclip", "-selection", "clipboard")
			case "xsel":
				cmd = exec.Command("xsel", "--clipboard", "--input")
			case "pbcopy":
				cmd = exec.Command("pbcopy")
			case "clip.exe":
				cmd = exec.Command("clip.exe")
			}
			break
		}
	}

	if cmd == nil {
		return fmt.Errorf("no clipboard tool found (tried: %v)", commands)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %v", err)
	}

	if _, err := stdin.Write([]byte(value)); err != nil {
		return fmt.Errorf("failed to write to stdin: %v", err)
	}
	if err := stdin.Close(); err != nil {
		return fmt.Errorf("failed to close stdin: %v", err)
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute %s: %v", cmd.Path, err)
	}

	return nil
}
