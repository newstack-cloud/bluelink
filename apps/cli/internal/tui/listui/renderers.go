package listui

import (
	"strconv"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
)

func renderStatus(status core.InstanceStatus, styles *stylespkg.Styles) string {
	switch status {
	case core.InstanceStatusDeployed:
		return styles.Success.Render("deployed")
	case core.InstanceStatusUpdated:
		return styles.Success.Render("updated")
	case core.InstanceStatusDeploying:
		return styles.Warning.Render("deploying")
	case core.InstanceStatusUpdating:
		return styles.Warning.Render("updating")
	case core.InstanceStatusDestroying:
		return styles.Warning.Render("destroying")
	case core.InstanceStatusDeployFailed:
		return styles.Error.Render("deploy failed")
	case core.InstanceStatusUpdateFailed:
		return styles.Error.Render("update failed")
	case core.InstanceStatusDestroyFailed:
		return styles.Error.Render("destroy failed")
	case core.InstanceStatusDestroyed:
		return styles.Muted.Render("destroyed")
	default:
		return styles.Muted.Render(status.String())
	}
}

func formatTimestamp(timestamp int64) string {
	if timestamp == 0 {
		return "Never"
	}
	t := time.Unix(timestamp, 0)
	return formatRelativeTime(t)
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "Just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return itoa(mins) + " minutes ago"
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return itoa(hours) + " hours ago"
	case diff < 48*time.Hour:
		return "Yesterday"
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return itoa(days) + " days ago"
	default:
		return t.Format("Jan 2, 2006")
	}
}

func padRight(s string, length int) string {
	if len(s) >= length {
		return s
	}
	return s + spaces(length-len(s))
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
