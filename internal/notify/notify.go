// Package notify posts a desktop notification banner.
package notify

import (
	"os/exec"
	"runtime"
	"strings"
)

// Send posts a notification. Best-effort: failures are swallowed so a
// missing helper never crashes the run loop.
func Send(title, subtitle, message string) {
	body := message
	if subtitle != "" {
		if body != "" {
			body = subtitle + " — " + body
		} else {
			body = subtitle
		}
	}
	switch runtime.GOOS {
	case "darwin":
		sendDarwin(title, subtitle, message)
	case "windows":
		sendWindows(title, body)
	default:
		sendLinux(title, body)
	}
}

func sendDarwin(title, subtitle, message string) {
	script := `display notification ` + quoteAS(message) + ` with title ` + quoteAS(title)
	if subtitle != "" {
		script += ` subtitle ` + quoteAS(subtitle)
	}
	_ = exec.Command("osascript", "-e", script).Run()
}

func quoteAS(s string) string {
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, `"`, `\"`)
	return `"` + s + `"`
}

func sendWindows(title, body string) {
	ps := `
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
$template = [Windows.UI.Notifications.ToastNotificationManager]::GetTemplateContent([Windows.UI.Notifications.ToastTemplateType]::ToastText02)
$text = $template.GetElementsByTagName('text')
$text[0].AppendChild($template.CreateTextNode(` + psQuote(title) + `)) | Out-Null
$text[1].AppendChild($template.CreateTextNode(` + psQuote(body) + `)) | Out-Null
$toast = [Windows.UI.Notifications.ToastNotification]::new($template)
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier('Tomatick').Show($toast)
`
	_ = exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", ps).Run()
}

func psQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

func sendLinux(title, body string) {
	if _, err := exec.LookPath("notify-send"); err == nil {
		_ = exec.Command("notify-send", "--app-name=Tomatick", title, body).Run()
	}
}
