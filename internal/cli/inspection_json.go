package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/graph"
	"github.com/mako10k/sealgraph/internal/history"
	"github.com/mako10k/sealgraph/internal/repository"
)

func extractInspectionFormat(args []string) ([]string, bool, error) {
	result := make([]string, 0, len(args))
	format := "human"
	seen := false
	for i := 0; i < len(args); i++ {
		value := args[i]
		if value != "--format" && !strings.HasPrefix(value, "--format=") {
			result = append(result, value)
			continue
		}
		if seen {
			return nil, false, fmt.Errorf("--format may be specified only once")
		}
		seen = true
		if value == "--format" {
			i++
			if i == len(args) {
				return nil, false, fmt.Errorf("--format requires human or json")
			}
			format = args[i]
		} else {
			format = strings.TrimPrefix(value, "--format=")
		}
		if format != "human" && format != "json" {
			return nil, false, fmt.Errorf("unsupported inspection format %q; expected human or json", format)
		}
	}
	return result, format == "json", nil
}

func writeInspectionJSON(stdout, stderr io.Writer, command string, value any) int {
	data, err := json.Marshal(value)
	if err != nil {
		return commandError(stderr, command, fmt.Errorf("encode JSON output: %w", err))
	}
	data = append(data, '\n')
	if _, err := stdout.Write(data); err != nil {
		return commandError(stderr, command, fmt.Errorf("write JSON output: %w", err))
	}
	return 0
}

func idValue(id *domain.ObjectID) any {
	if id == nil {
		return nil
	}
	return id.String()
}
func contentJSON(ref domain.ContentRef) map[string]any {
	return map[string]any{"store": ref.Store, "type": ref.Type, "object_id": ref.ID.String()}
}
func attachmentJSON(value domain.Attachment) map[string]any {
	return map[string]any{"name": value.Name, "media_type": value.MediaType, "blob": contentJSON(value.Blob)}
}
func linkJSON(value domain.Link) map[string]any {
	return map[string]any{"target_seal_id": value.TargetSeal.String(), "message": value.Message}
}
func payloadJSON(payload domain.SealPayload) map[string]any {
	attachments := make([]any, 0, len(payload.Attachments))
	for _, value := range payload.Attachments {
		attachments = append(attachments, attachmentJSON(value))
	}
	links := make([]any, 0, len(payload.Links))
	for _, value := range payload.Links {
		links = append(links, linkJSON(value))
	}
	return map[string]any{"parent_revision": idValue(payload.ParentRevision), "content": contentJSON(payload.Content), "root": payload.Root, "draft": payload.Draft, "attachments": attachments, "links": links}
}

func showJSON(result repository.ShowResult) map[string]any {
	value := payloadJSON(result.Payload)
	value["schema"] = "sealgraph/show/v1"
	value["seal_id"] = result.ID.String()
	value["current_refs"] = stringsOrEmpty(result.REFNames)
	value["content_bytes"] = len(result.Content)
	return value
}

func stringsOrEmpty(values []string) []string { return append([]string{}, values...) }

func statusJSON(value repository.RefStatus) map[string]any {
	direct := make([]string, 0, len(value.StaleDirect))
	for _, id := range value.StaleDirect {
		direct = append(direct, id.String())
	}
	transitive := make([][]string, 0, len(value.StaleTransitive))
	for _, path := range value.StaleTransitive {
		ids := make([]string, 0, len(path))
		for _, id := range path {
			ids = append(ids, id.String())
		}
		transitive = append(transitive, ids)
	}
	candidateRelation := "NO_CANDIDATE"
	if value.Unsealed {
		candidateRelation = "UNSEALED"
	}
	var source any
	if value.Source != nil {
		source = map[string]any{"path": value.Source.Path, "baseline": value.Source.Baseline, "relation": value.Source.Relation}
	}
	return map[string]any{"ref": value.REF, "head_seal_id": idValue(value.Head), "candidate_to_head": candidateRelation, "draft": value.Draft, "stale": map[string]any{"self": value.StaleSelf, "direct_target_seal_ids": direct, "transitive_paths": transitive}, "sealed_state_labels": sealedStatusLabels(value.Labels()), "local_source": source}
}

func sourceCompareJSON(value repository.SourceCompareResult) map[string]any {
	var baseline any
	if value.BaselineContent != nil {
		baseline = contentJSON(*value.BaselineContent)
	}
	return map[string]any{"schema": "sealgraph/source-compare/v1", "ref": value.REF, "path": value.Path, "baseline": value.Baseline, "baseline_content": baseline, "workfile_content": map[string]any{"store": domain.NativeStore, "type": domain.BlobType, "object_id": value.WorkfileID.String(), "bytes": value.WorkfileBytes}, "relation": value.Relation}
}

func sealedStatusLabels(labels []string) []string {
	result := append([]string(nil), labels...)
	for i := range result {
		if result[i] == "CLEAN" {
			result[i] = "SEALED_STATE_CLEAN"
		}
	}
	return result
}
func statusesJSON(schema string, statuses []repository.RefStatus, extra map[string]any) map[string]any {
	items := make([]any, 0, len(statuses))
	for _, status := range statuses {
		items = append(items, statusJSON(status))
	}
	result := map[string]any{"schema": schema, "statuses": items}
	for key, value := range extra {
		result[key] = value
	}
	return result
}

func graphJSON(nodes []repository.GraphNode) map[string]any {
	items := make([]any, 0, len(nodes))
	for _, node := range nodes {
		links := make([]any, 0, len(node.Links))
		for _, link := range node.Links {
			links = append(links, map[string]any{"target_seal_id": link.Target.String(), "state": string(link.State)})
		}
		items = append(items, map[string]any{"seal_id": node.ID.String(), "state": string(node.State), "refs": stringsOrEmpty(node.REFs), "parent_revision": idValue(node.Parent), "causes": links})
	}
	return map[string]any{"schema": "sealgraph/graph/v1", "nodes": items}
}
func impactJSON(source domain.ObjectID, impacts []graph.Impact, allPaths bool, limit int) map[string]any {
	items := make([]any, 0, len(impacts))
	for _, impact := range impacts {
		paths := make([][]string, 0, len(impact.Paths))
		for _, path := range impact.Paths {
			ids := make([]string, 0, len(path))
			for _, id := range path {
				ids = append(ids, id.String())
			}
			paths = append(paths, ids)
		}
		items = append(items, map[string]any{"head_seal_id": impact.Head.String(), "refs": stringsOrEmpty(impact.REFs), "paths": paths, "paths_truncated": impact.Truncated})
	}
	result := map[string]any{"schema": "sealgraph/impact/v1", "source_seal_id": source.String(), "all_paths": allPaths, "impacts": items}
	if allPaths {
		result["max_paths"] = limit
	} else {
		result["max_paths"] = nil
	}
	return result
}
func entryJSON(entry history.Entry) map[string]any {
	value := payloadJSON(entry.Payload)
	value["seal_id"] = entry.ID.String()
	return value
}
func logJSON(ref string, entries []history.Entry) map[string]any {
	items := make([]any, 0, len(entries))
	for _, entry := range entries {
		items = append(items, entryJSON(entry))
	}
	return map[string]any{"schema": "sealgraph/log/v1", "ref": ref, "entries": items}
}

func linkChangeJSON(change history.LinkChange) map[string]any {
	return map[string]any{"kind": string(change.Kind), "target_seal_id": change.TargetSeal.String(), "before_seal_id": idValue(change.BeforeSeal), "after_seal_id": idValue(change.AfterSeal), "before_message": change.BeforeMessage, "after_message": change.AfterMessage}
}
func linkLogJSON(ref, upstream string, entries []history.LinkLogEntry) map[string]any {
	items := make([]any, 0, len(entries))
	for _, entry := range entries {
		changes := make([]any, 0, len(entry.Changes))
		for _, change := range entry.Changes {
			changes = append(changes, linkChangeJSON(change))
		}
		items = append(items, map[string]any{"seal_id": entry.Entry.ID.String(), "parent_revision": idValue(entry.Entry.Payload.ParentRevision), "changes": changes})
	}
	var target any
	if upstream != "" {
		target = upstream
	}
	return map[string]any{"schema": "sealgraph/linklog/v1", "ref": ref, "upstream_seal_id": target, "entries": items}
}

func attachmentChangeJSON(change history.AttachmentChangeRecord) map[string]any {
	var before, after any
	if change.Before != nil {
		before = attachmentJSON(*change.Before)
	}
	if change.After != nil {
		after = attachmentJSON(*change.After)
	}
	return map[string]any{"kind": string(change.Kind), "name": change.Name, "before": before, "after": after}
}
func compareJSON(diff history.SealDiff) map[string]any {
	attachments := make([]any, 0, len(diff.Attachments))
	for _, change := range diff.Attachments {
		attachments = append(attachments, attachmentChangeJSON(change))
	}
	links := make([]any, 0, len(diff.Links))
	for _, change := range diff.Links {
		links = append(links, linkChangeJSON(change))
	}
	return map[string]any{"schema": "sealgraph/compare/v1", "from_seal_id": diff.From.String(), "to_seal_id": diff.To.String(), "content": map[string]any{"changed": diff.Content.Changed, "before": contentJSON(diff.Content.Before), "after": contentJSON(diff.Content.After)}, "attachments": attachments, "links": links, "root": map[string]any{"changed": diff.Root.Changed, "before": diff.Root.Before, "after": diff.Root.After}, "draft": map[string]any{"changed": diff.Draft.Changed, "before": diff.Draft.Before, "after": diff.Draft.After}, "parent_revision": map[string]any{"changed": diff.Parent.Changed, "before": idValue(diff.Parent.Before), "after": idValue(diff.Parent.After)}}
}

func fsckJSON(report repository.FsckReport) map[string]any {
	detached := make([]string, 0, len(report.HistoricalOrDetachedSeals))
	for _, id := range report.HistoricalOrDetachedSeals {
		detached = append(detached, id.String())
	}
	unreferenced := make([]string, 0, len(report.UnreferencedObjects))
	for _, id := range report.UnreferencedObjects {
		unreferenced = append(unreferenced, id.String())
	}
	return map[string]any{"schema": "sealgraph/fsck/v1", "result": "ok", "objects": report.Objects, "seals": report.Seals, "material_objects": report.MaterialObjects, "refs": report.REFs, "tags": report.Tags, "active_seals": report.ActiveSeals, "historical_or_detached_seal_ids": detached, "unreferenced_object_ids": unreferenced}
}
