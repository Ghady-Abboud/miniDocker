package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

const alpineRootFS = "tmp/rootfs/"

func main() {
	if len(os.Args) > 1 && os.Args[1] == "child" {
		if err := runChild(); err != nil {
			fmt.Println("Error in child:", err)
			os.Exit(1)
		}
		return
	}
	runParent()
}

func runChild() error {
	if err := syscall.Mount("", "/", "", syscall.MS_PRIVATE | syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("make mounts private: %w", err)
	}

	resolvTarget := alpineRootFS + "etc/resolv.conf"
	if err := os.WriteFile(resolvTarget, []byte{}, 0644); err != nil {
		return fmt.Errorf("create resolv.conf: %w", err)
	}
	if err := syscall.Mount("/etc/resolv.conf", resolvTarget, "", syscall.MS_BIND, ""); err != nil {
		return fmt.Errorf("bind mount resolv.conf: %w", err)
	}

	if err := isolateRootFS(alpineRootFS); err != nil {
		return fmt.Errorf("isolate rootFS: %w", err)
	}

	if err := syscall.Mount("proc", "/proc", "proc", 0, ""); err != nil{
		return fmt.Errorf("mount proc: %w", err)
	}

	return syscall.Exec("/bin/sh", []string{"/bin/sh"}, os.Environ())
}

func runParent() {
	cmd := exec.Command("/proc/self/exe", "child")

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Cloneflags: syscall.CLONE_NEWPID | syscall.CLONE_NEWUTS | syscall.CLONE_NEWNS,
	}

	fmt.Print("Container created successfully!\n")

	if err := cmd.Start(); err != nil {
		fmt.Println("Error starting container:", err)
		return
	}

	if err := setupCgroup(cmd.Process.Pid); err != nil {
		fmt.Println("Error setting up cgroup:", err)
		return
	}

	if err := cmd.Wait(); err != nil {
		fmt.Println("Error waiting for container:", err)
		return
	}
}

func isolateRootFS(rootFS string) error {
	oldRootFS := "/oldRootFS"

	if err := syscall.Mount(rootFS, rootFS, "", syscall.MS_BIND | syscall.MS_REC, ""); err != nil {
		return fmt.Errorf("bind mount rootFS: %w", err)
	}

	if err := os.Mkdir(rootFS + oldRootFS, 0755); err != nil && !os.IsExist(err) {
		return fmt.Errorf("create oldRootFS dir: %w", err)
	}

	if err := syscall.PivotRoot(rootFS, rootFS + oldRootFS); err != nil {
		return fmt.Errorf("pivot root: %w", err)
	}

	if err := os.Chdir("/"); err != nil {
		return fmt.Errorf("change dir to new root: %w", err)
	}

	if err := syscall.Unmount(oldRootFS, syscall.MNT_DETACH); err != nil {
		return fmt.Errorf("unmount old rootFS: %w", err)
	}

	if err := os.Remove(oldRootFS); err != nil {
		return fmt.Errorf("remove oldRootFS dir: %w", err)
	}

	return nil
}

func setupCgroup(pid int) error {
	cgroupPath := "/sys/fs/cgroup"
	cgroupName := "miniDock"
	miniDockCgroupPath := filepath.Join(cgroupPath, cgroupName)

	if err := os.WriteFile(cgroupPath+"/cgroup.subtree_control", []byte("+memory +cpu"), 0644); err != nil {
		return fmt.Errorf("enable cgroup controllers: %w", err)
	}

	if err := os.Mkdir(miniDockCgroupPath, 0755); err != nil && !os.IsExist(err) {
		fmt.Printf("create cgroup: %v\n", err)
	}

	memLimit := filepath.Join(miniDockCgroupPath, "memory.max")
	if err := os.WriteFile(memLimit, []byte("52428800"), 0644); err != nil {
		return fmt.Errorf("set memory limit: %w", err)
	}

	cpuQuota := filepath.Join(miniDockCgroupPath, "cpu.max")
	if err := os.WriteFile(cpuQuota, []byte("20000 100000"), 0644); err != nil {
		return fmt.Errorf("set cpu quota: %w", err)
	}

	cpuProcs := filepath.Join(miniDockCgroupPath, "cgroup.procs")
	if err := os.WriteFile(cpuProcs, []byte(strconv.Itoa(pid)), 0644); err != nil {
		return fmt.Errorf("add process to cgroup: %w", err)
	}

	fmt.Printf("Cgroup: PID %d limited to 50MB memory and 20%% CPU\n", pid)
	return nil
}
