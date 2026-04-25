package cmd

import (
	"fmt"
	"os/exec"
	"strings"
)

var notificationCommandRunner = runNotificationCommand

func notifyPipelineError(summary string, err error) {
	if err == nil {
		return
	}
	message := trimForNotification(err.Error(), 400)
	_ = notificationCommandRunner("notify-send", "iNiR Theme", fmt.Sprintf("%s\n%s", summary, message), "-a", "inir-cli")
}

func notifyApplyFailures(failures []targetFailure) {
	if len(failures) == 0 {
		return
	}
	maxItems := len(failures)
	if maxItems > 3 {
		maxItems = 3
	}
	parts := make([]string, 0, maxItems)
	for _, failure := range failures[:maxItems] {
		parts = append(parts, fmt.Sprintf("%s: %s", failure.Target, trimForNotification(failure.Error, 100)))
	}
	if len(failures) > maxItems {
		parts = append(parts, fmt.Sprintf("+%d more", len(failures)-maxItems))
	}
	_ = notificationCommandRunner("notify-send", "iNiR Theme Apply", strings.Join(parts, " | "), "-a", "inir-cli")
}

func runNotificationCommand(name string, args ...string) error {
	if _, err := exec.LookPath(name); err != nil {
		return nil
	}
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func trimForNotification(input string, maxLen int) string {
	if len(input) <= maxLen {
		return input
	}
	if maxLen <= 3 {
		return input[:maxLen]
	}
	return input[:maxLen-3] + "..."
}
