package output

import (
	"bytes"
	"fmt"
	"html/template"
	"sort"
	"time"

	"github.com/raesene/eolas/pkg/storage"
)

// TimelineFormatter generates timeline-based HTML reports
type TimelineFormatter struct {
	template *template.Template
}

// TimelineData represents the data for timeline reports
type TimelineData struct {
	Title            string
	GeneratedAt      string
	ConfigName       string
	TotalVersions    int
	TimeSpan         string
	Timeline         []TimelineEntry
	ResourceTrends   []ResourceTrend
	SecurityTrends   []SecurityTrend
	CurrentSnapshot  SnapshotData
	PreviousSnapshot *SnapshotData
}

// TimelineEntry represents a single point in the configuration timeline
type TimelineEntry struct {
	ID               string
	Timestamp        time.Time
	FormattedTime    string
	TotalResources   int
	SecurityIssues   int
	ResourceChanges  map[string]int
	SecurityChanges  map[string]int
	IsLatest         bool
}

// ResourceTrend represents how a resource type has changed over time
type ResourceTrend struct {
	ResourceType string
	DataPoints   []TrendPoint
	TotalChange  int
	Trend        string // "increasing", "decreasing", "stable"
}

// SecurityTrend represents how security findings have changed over time
type SecurityTrend struct {
	FindingType string
	DataPoints  []TrendPoint
	TotalChange int
	Trend       string
}

// TrendPoint represents a data point in a trend
type TrendPoint struct {
	Timestamp time.Time
	Value     int
}

// SnapshotData represents a configuration snapshot
type SnapshotData struct {
	ID                    string
	Timestamp             time.Time
	FormattedTime         string
	ResourceCounts        map[string]int
	TotalResources        int
	PrivilegedContainers  int
	CapabilityContainers  int
	HostNamespaceUsage    int
	HostPathVolumes       int
	TotalSecurityIssues   int
}

// NewTimelineFormatter creates a new timeline HTML formatter
func NewTimelineFormatter() (*TimelineFormatter, error) {
	tmpl, err := template.New("timeline").Funcs(template.FuncMap{
		"add": func(a, b int) int { return a + b },
		"formatChange": func(change int) string {
			if change > 0 {
				return fmt.Sprintf("+%d", change)
			}
			return fmt.Sprintf("%d", change)
		},
		"changeClass": func(change int) string {
			if change > 0 {
				return "positive"
			} else if change < 0 {
				return "negative"
			}
			return "neutral"
		},
		"trendClass": func(trend string) string {
			switch trend {
			case "increasing":
				return "trend-up"
			case "decreasing":
				return "trend-down"
			default:
				return "trend-stable"
			}
		},
	}).Parse(timelineTemplate)
	
	if err != nil {
		return nil, fmt.Errorf("failed to parse timeline template: %w", err)
	}

	return &TimelineFormatter{
		template: tmpl,
	}, nil
}

// GenerateTimelineHTML creates a timeline-based HTML report
func (f *TimelineFormatter) GenerateTimelineHTML(configName string, history []storage.ConfigMetadata, securityHistory []storage.StoredSecurityAnalysis) ([]byte, error) {
	if len(history) == 0 {
		return nil, fmt.Errorf("no configuration history available")
	}

	// Sort history by timestamp
	sort.Slice(history, func(i, j int) bool {
		return history[i].Timestamp.Before(history[j].Timestamp)
	})

	// Calculate time span
	timeSpan := ""
	if len(history) > 1 {
		start := history[0].Timestamp
		end := history[len(history)-1].Timestamp
		timeSpan = fmt.Sprintf("%s to %s", start.Format("Jan 2, 2006"), end.Format("Jan 2, 2006"))
	} else {
		timeSpan = history[0].Timestamp.Format("Jan 2, 2006")
	}

	// Build timeline entries
	timeline := make([]TimelineEntry, len(history))
	securityMap := make(map[string]storage.StoredSecurityAnalysis)
	
	for _, sec := range securityHistory {
		securityMap[sec.ConfigID] = sec
	}

	for i, config := range history {
		securityCount := 0
		if sec, exists := securityMap[config.ID]; exists {
			securityCount = len(sec.PrivilegedContainers) + len(sec.CapabilityContainers) + 
			               len(sec.HostNamespaceWorkloads) + len(sec.HostPathVolumes)
		}

		totalResources := 0
		for _, count := range config.ResourceCounts {
			totalResources += count
		}

		timeline[i] = TimelineEntry{
			ID:              config.ID,
			Timestamp:       config.Timestamp,
			FormattedTime:   config.Timestamp.Format("Jan 2, 15:04"),
			TotalResources:  totalResources,
			SecurityIssues:  securityCount,
			IsLatest:        i == len(history)-1,
		}

		// Calculate changes from previous version
		if i > 0 {
			prevConfig := history[i-1]
			resourceChanges := make(map[string]int)
			
			// Get all resource types from both versions
			allTypes := make(map[string]bool)
			for rt := range config.ResourceCounts {
				allTypes[rt] = true
			}
			for rt := range prevConfig.ResourceCounts {
				allTypes[rt] = true
			}

			for rt := range allTypes {
				current := config.ResourceCounts[rt]
				previous := prevConfig.ResourceCounts[rt]
				if current != previous {
					resourceChanges[rt] = current - previous
				}
			}
			timeline[i].ResourceChanges = resourceChanges

			// Calculate security changes
			prevSecurityCount := 0
			if prevSec, exists := securityMap[prevConfig.ID]; exists {
				prevSecurityCount = len(prevSec.PrivilegedContainers) + len(prevSec.CapabilityContainers) + 
				                   len(prevSec.HostNamespaceWorkloads) + len(prevSec.HostPathVolumes)
			}

			if securityCount != prevSecurityCount {
				timeline[i].SecurityChanges = map[string]int{
					"total": securityCount - prevSecurityCount,
				}
			}
		}
	}

	// Build resource trends
	resourceTrends := f.buildResourceTrends(history)
	securityTrends := f.buildSecurityTrends(history, securityHistory)

	// Build snapshots
	latest := history[len(history)-1]
	currentSnapshot := f.buildSnapshot(latest, securityMap[latest.ID])
	
	var previousSnapshot *SnapshotData
	if len(history) > 1 {
		prev := history[len(history)-2]
		prevSnap := f.buildSnapshot(prev, securityMap[prev.ID])
		previousSnapshot = &prevSnap
	}

	data := TimelineData{
		Title:            "Configuration Timeline Report - " + configName,
		GeneratedAt:      time.Now().Format(time.RFC1123),
		ConfigName:       configName,
		TotalVersions:    len(history),
		TimeSpan:         timeSpan,
		Timeline:         timeline,
		ResourceTrends:   resourceTrends,
		SecurityTrends:   securityTrends,
		CurrentSnapshot:  currentSnapshot,
		PreviousSnapshot: previousSnapshot,
	}

	var buf bytes.Buffer
	if err := f.template.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute timeline template: %w", err)
	}

	return buf.Bytes(), nil
}

// buildResourceTrends creates trend data for resources over time
func (f *TimelineFormatter) buildResourceTrends(history []storage.ConfigMetadata) []ResourceTrend {
	if len(history) < 2 {
		return nil
	}

	// Get all resource types
	allTypes := make(map[string]bool)
	for _, config := range history {
		for rt := range config.ResourceCounts {
			allTypes[rt] = true
		}
	}

	var trends []ResourceTrend
	for rt := range allTypes {
		var dataPoints []TrendPoint
		first := 0
		last := 0
		
		for _, config := range history {
			count := config.ResourceCounts[rt]
			dataPoints = append(dataPoints, TrendPoint{
				Timestamp: config.Timestamp,
				Value:     count,
			})
			
			if len(dataPoints) == 1 {
				first = count
			}
			last = count
		}

		totalChange := last - first
		trend := "stable"
		if totalChange > 0 {
			trend = "increasing"
		} else if totalChange < 0 {
			trend = "decreasing"
		}

		// Only include trends with significant changes
		if totalChange != 0 {
			trends = append(trends, ResourceTrend{
				ResourceType: rt,
				DataPoints:   dataPoints,
				TotalChange:  totalChange,
				Trend:        trend,
			})
		}
	}

	// Sort by magnitude of change
	sort.Slice(trends, func(i, j int) bool {
		return abs(trends[i].TotalChange) > abs(trends[j].TotalChange)
	})

	// Limit to top 10 most significant trends
	if len(trends) > 10 {
		trends = trends[:10]
	}

	return trends
}

// buildSecurityTrends creates trend data for security findings over time
func (f *TimelineFormatter) buildSecurityTrends(history []storage.ConfigMetadata, securityHistory []storage.StoredSecurityAnalysis) []SecurityTrend {
	if len(history) < 2 {
		return nil
	}

	securityMap := make(map[string]storage.StoredSecurityAnalysis)
	for _, sec := range securityHistory {
		securityMap[sec.ConfigID] = sec
	}

	findingTypes := []string{
		"Privileged Containers",
		"Capability Containers", 
		"Host Namespace Usage",
		"Host Path Volumes",
	}

	var trends []SecurityTrend
	for _, findingType := range findingTypes {
		var dataPoints []TrendPoint
		first := 0
		last := 0

		for _, config := range history {
			count := 0
			if sec, exists := securityMap[config.ID]; exists {
				switch findingType {
				case "Privileged Containers":
					count = len(sec.PrivilegedContainers)
				case "Capability Containers":
					count = len(sec.CapabilityContainers)
				case "Host Namespace Usage":
					count = len(sec.HostNamespaceWorkloads)
				case "Host Path Volumes":
					count = len(sec.HostPathVolumes)
				}
			}

			dataPoints = append(dataPoints, TrendPoint{
				Timestamp: config.Timestamp,
				Value:     count,
			})

			if len(dataPoints) == 1 {
				first = count
			}
			last = count
		}

		totalChange := last - first
		trend := "stable"
		if totalChange > 0 {
			trend = "increasing"
		} else if totalChange < 0 {
			trend = "decreasing"
		}

		trends = append(trends, SecurityTrend{
			FindingType: findingType,
			DataPoints:  dataPoints,
			TotalChange: totalChange,
			Trend:       trend,
		})
	}

	return trends
}

// buildSnapshot creates a snapshot from configuration metadata
func (f *TimelineFormatter) buildSnapshot(config storage.ConfigMetadata, security storage.StoredSecurityAnalysis) SnapshotData {
	totalResources := 0
	for _, count := range config.ResourceCounts {
		totalResources += count
	}

	privileged := len(security.PrivilegedContainers)
	capability := len(security.CapabilityContainers)
	hostNS := len(security.HostNamespaceWorkloads)
	hostPath := len(security.HostPathVolumes)

	return SnapshotData{
		ID:                   config.ID,
		Timestamp:            config.Timestamp,
		FormattedTime:        config.Timestamp.Format("Jan 2, 2006 15:04"),
		ResourceCounts:       config.ResourceCounts,
		TotalResources:       totalResources,
		PrivilegedContainers: privileged,
		CapabilityContainers: capability,
		HostNamespaceUsage:   hostNS,
		HostPathVolumes:      hostPath,
		TotalSecurityIssues:  privileged + capability + hostNS + hostPath,
	}
}

// abs returns the absolute value of an integer
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}