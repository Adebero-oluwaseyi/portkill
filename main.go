package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

func main() {
	force := flag.Bool("f", false, "force kill without asking for confirmation")
	flag.Parse()

	if flag.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "usage: portkill [-f] <port>")
		os.Exit(1)
	}

	port, err := strconv.Atoi(flag.Arg(0))
	if err != nil || port < 1 || port > 65535 {
		fmt.Fprintf(os.Stderr, "error: invalid port %q\n", flag.Arg(0))
		os.Exit(1)
	}

	pids, err := findPIDsByPort(port)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	if len(pids) == 0 {
		fmt.Printf("no process found listening on port %d\n", port)
		return
	}

	for _, pid := range pids {
		name := getProcessName(pid)
		if !*force {
			fmt.Printf("kill %s (PID %d) on port %d? [y/N]: ", name, pid, port)
			reader := bufio.NewReader(os.Stdin)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				fmt.Printf("skipped PID %d\n", pid)
				continue
			}
		}

		if err := killProcess(pid); err != nil {
			fmt.Fprintf(os.Stderr, "failed to kill PID %d: %v\n", pid, err)
			continue
		}
		fmt.Printf("killed %s (PID %d) on port %d\n", name, pid, port)
	}
}

func findPIDsByPort(port int) ([]int, error) {
	seen := make(map[int]bool)
	var pids []int
	portSuffix := fmt.Sprintf(":%d", port)

	if runtime.GOOS == "windows" {
		out, err := exec.Command("netstat", "-ano").Output()
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(strings.NewReader(string(out)))
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 5 {
				continue
			}
			// fields: [Proto, Local Address, Foreign Address, State, PID]
			if strings.HasSuffix(fields[1], portSuffix) && fields[3] == "LISTENING" {
				pid, err := strconv.Atoi(fields[4])
				if err == nil && pid > 0 && !seen[pid] {
					seen[pid] = true
					pids = append(pids, pid)
				}
			}
		}
		return pids, nil
	}

	// macOS / Linux
	out, err := exec.Command("lsof", "-t", "-i", portSuffix).Output()
	if err != nil {
		// lsof exits with code 1 when no matching process is found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}

	for _, line := range strings.Fields(string(out)) {
		pid, err := strconv.Atoi(line)
		if err == nil && pid > 0 && !seen[pid] {
			seen[pid] = true
			pids = append(pids, pid)
		}
	}
	return pids, nil
}

func getProcessName(pid int) string {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/FO", "CSV", "/NH").Output()
		if err == nil {
			parts := strings.Split(strings.TrimSpace(string(out)), ",")
			if len(parts) > 0 {
				return strings.Trim(parts[0], "\"")
			}
		}
		return "unknown"
	}

	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "comm=").Output()
	if err == nil {
		name := strings.TrimSpace(string(out))
		if name != "" {
			return name
		}
	}
	return "unknown"
}

func killProcess(pid int) error {
	if runtime.GOOS == "windows" {
		return exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/F").Run()
	}
	return exec.Command("kill", "-9", strconv.Itoa(pid)).Run()
}
