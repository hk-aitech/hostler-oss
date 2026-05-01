// trac: TRC-CM004,TRC-QR004
package cmd

import (
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/brand"
	"os"

	"github.com/spf13/cobra"

	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/app"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/internal/ports"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/apperr"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/audit"
	"github.com/hk-aitech/hostler-oss/packages/hostler-cli/pkg/db"
)

// auditCmd is the root of the audit subcommand.
var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Query the audit log",
	RunE:  runAuditQuery,
}

var (
	auditEntityType string
	auditEntityID   string
	auditEventType  string
	auditSince      string
	auditUntil      string
	auditLimit      int
	auditLast       int // explicit time-desc N flag
)

func runAuditQuery(cmd *cobra.Command, args []string) error {
	mustInitDB()
	defer db.Close()

	// When --last and --limit are both specified, --last wins (per SSOT).
	// audit already orders by timestamp DESC, matching --last semantics, so
	// we just overwrite the limit value with last.
	effectiveLimit := auditLimit
	if auditLast > 0 {
		effectiveLimit = auditLast
	}

	filter := ports.AuditQueryFilter{
		EntityType: auditEntityType,
		EntityID:   auditEntityID,
		EventType:  auditEventType,
		Since:      auditSince,
		Until:      auditUntil,
		Limit:      effectiveLimit,
	}

	events, err := app.AuditQuery(filter)
	if err != nil {
		Out.Error("failed to query audit log: "+err.Error(), "", "")
		os.Exit(exitError)
	}

	if outputFormat == "json" {
		Out.Print(events)
		return nil
	}

	headers := []string{"ID", "Timestamp", "EventType", "EntityType", "EntityID", "Actor", "Session"}
	rows := make([][]string, 0, len(events))
	for _, e := range events {
		rows = append(rows, []string{
			itoa64(e.ID),
			e.Timestamp,
			e.EventType,
			e.EntityType,
			e.EntityID,
			e.ActorID,
			e.SessionID,
		})
	}
	Out.Table(headers, rows)
	return nil
}

// itoa64 converts an int64 to its string form.
func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	buf := make([]byte, 0, 20)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

// ---------------------------------------------------------------------------
// init
// ---------------------------------------------------------------------------

func init() {
	auditCmd.Flags().StringVar(&auditEntityType, "entity-type", "", "Entity type filter (e.g., task, sprint)")
	auditCmd.Flags().StringVar(&auditEntityID, "entity-id", "", "Entity ID filter (e.g., T001)")
	auditCmd.Flags().StringVar(&auditEventType, "event-type", "", "Event type filter (e.g., task.created)")
	auditCmd.Flags().StringVar(&auditSince, "since", "", "Start timestamp ISO string (e.g., 2026-01-01)")
	auditCmd.Flags().StringVar(&auditUntil, "until", "", "End timestamp ISO string (e.g., 2026-12-31)")
	auditCmd.Flags().IntVar(&auditLimit, "limit", 50, "Maximum number of rows to return (default: 50)")
	// --last flag - matches SSOT. audit already orders by timestamp DESC, so
	// the semantics are identical, but the flag spells out the intent.
	auditCmd.Flags().IntVar(&auditLast, "last", 0, "Most recent N rows (timestamp DESC - wins when used with --limit)")

	auditCmd.AddCommand(auditRestoreCmd)
	rootCmd.AddCommand(auditCmd)
}

// ---------------------------------------------------------------------------
// audit restore - reconstruct audit_events from git log
// ---------------------------------------------------------------------------

var auditRestoreApply bool

var auditRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore sprint.completed / task.completed events from git log",
	Long: `Reconstruct missing audit_events using the first commit timestamp from
git log for works/sprints/completed/*/SPRINT.md and tasks/T*.md.

Defaults to dry-run - no DB changes when --apply is omitted.
Natural-key dedupe - skip if an event with the same
(event_type, entity_type, entity_id) already exists. Existing event
timestamps are never overwritten.

Useful for filling in history lost during a manual DB drift merge.`,
	Example: brand.Examplef(
		"audit restore               # dry-run",
		"audit restore --apply -o json | jq '.data.restored'",
	),
	RunE: func(cmd *cobra.Command, args []string) error {
		mustInitDB()
		defer db.Close()

		raw, err := app.AuditRestore(!auditRestoreApply)
		if err != nil {
			Out.Error(err.Error(), "", "")
			os.Exit(exitError)
		}
		result, ok := raw.(*audit.RestoreResult)
		if !ok {
			Out.Error("failed to cast audit.RestoreResult", apperr.CategoryInvalidState.String(), "")
			os.Exit(exitError)
		}
		var msg string
		if result.DryRun {
			msg = "[dry-run] audit restore candidates aggregated - no writes performed"
		} else {
			msg = "audit restore complete"
		}
		Out.Success(msg, result)
		return nil
	},
}

func init() {
	auditRestoreCmd.Flags().BoolVar(&auditRestoreApply, "apply", false, "Actually INSERT into the DB (default: dry-run)")
}
