package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func MirrorClone(url, dest string) error {
	if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", "--mirror", url, dest)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// func MirrorPush(localPath, remoteURL string) error {

// 	cmd := exec.Command("git",
// 		"--git-dir", localPath,
// 		"push",
// 		"--mirror",
// 		remoteURL,
// 	)

// 	cmd.Stdout = os.Stdout
// 	cmd.Stderr = os.Stderr

// 	return cmd.Run()
// }

func PushAllAndTags(localPath, remoteURL string) error {

	pushAll := exec.Command(
		"git",
		"--git-dir", localPath,
		"push",
		"--all",
		remoteURL,
	)

	pushAll.Stdout = os.Stdout
	pushAll.Stderr = os.Stderr

	if err := pushAll.Run(); err != nil {
		return err
	}

	pushTags := exec.Command(
		"git",
		"--git-dir", localPath,
		"push",
		"--tags",
		remoteURL,
	)

	pushTags.Stdout = os.Stdout
	pushTags.Stderr = os.Stderr

	return pushTags.Run()
}

func MirrorPush(mirrorDir, targetRemoteURL string) error {
	// Ensure mirrorDir is a bare repo
	cmd := exec.Command("git", "--git-dir", mirrorDir, "remote", "remove", "target")
	_ = cmd.Run() // ignore

	cmd = exec.Command("git", "--git-dir", mirrorDir, "remote", "add", "target", targetRemoteURL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git remote add failed: %w\n%s", err, string(out))
	}

	cmd = exec.Command("git", "--git-dir", mirrorDir, "push", "--mirror", "target")
	out, err = cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git push --mirror failed: %w\n%s", err, string(out))
	}
	return nil
}
