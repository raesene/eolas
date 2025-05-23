package output

// timelineTemplate is the HTML template for timeline reports
const timelineTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{ .Title }}</title>
    <style>
        :root {
            --primary-color: #3498db;
            --secondary-color: #2c3e50;
            --background-color: #f8f9fa;
            --text-color: #333;
            --border-color: #ddd;
            --success-color: #27ae60;
            --warning-color: #f39c12;
            --danger-color: #e74c3c;
            --info-color: #3498db;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background-color: var(--background-color);
            color: var(--text-color);
            line-height: 1.6;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 20px;
        }

        .header {
            text-align: center;
            margin-bottom: 40px;
            padding: 30px;
            background: white;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }

        .header h1 {
            color: var(--primary-color);
            margin-bottom: 10px;
            font-size: 2.5em;
        }

        .header .subtitle {
            color: #666;
            font-size: 1.2em;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }

        .stat-card {
            background: white;
            padding: 25px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            text-align: center;
            border-left: 4px solid var(--primary-color);
        }

        .stat-card.danger {
            border-left-color: var(--danger-color);
        }

        .stat-card.warning {
            border-left-color: var(--warning-color);
        }

        .stat-card.success {
            border-left-color: var(--success-color);
        }

        .stat-number {
            font-size: 2.5em;
            font-weight: bold;
            color: var(--primary-color);
            display: block;
        }

        .stat-label {
            color: #666;
            margin-top: 5px;
            font-size: 0.9em;
        }

        .section {
            background: white;
            margin-bottom: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
            overflow: hidden;
        }

        .section-header {
            background: var(--primary-color);
            color: white;
            padding: 20px;
            font-size: 1.3em;
            font-weight: bold;
        }

        .section-content {
            padding: 25px;
        }

        .timeline {
            position: relative;
            padding-left: 30px;
        }

        .timeline::before {
            content: '';
            position: absolute;
            left: 15px;
            top: 0;
            bottom: 0;
            width: 2px;
            background: var(--border-color);
        }

        .timeline-entry {
            position: relative;
            margin-bottom: 30px;
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
            border-left: 4px solid var(--info-color);
        }

        .timeline-entry.latest {
            border-left-color: var(--success-color);
            background: #e8f5e8;
        }

        .timeline-entry::before {
            content: '';
            position: absolute;
            left: -37px;
            top: 25px;
            width: 12px;
            height: 12px;
            border-radius: 50%;
            background: var(--info-color);
            border: 3px solid white;
        }

        .timeline-entry.latest::before {
            background: var(--success-color);
        }

        .timeline-time {
            font-weight: bold;
            color: var(--primary-color);
            margin-bottom: 10px;
        }

        .timeline-stats {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
            gap: 15px;
            margin-bottom: 15px;
        }

        .timeline-stat {
            text-align: center;
            padding: 10px;
            background: white;
            border-radius: 5px;
        }

        .timeline-stat-value {
            font-size: 1.5em;
            font-weight: bold;
            color: var(--primary-color);
        }

        .timeline-stat-label {
            font-size: 0.8em;
            color: #666;
        }

        .changes {
            margin-top: 15px;
        }

        .change-item {
            display: inline-block;
            margin: 3px;
            padding: 5px 10px;
            border-radius: 15px;
            font-size: 0.85em;
            font-weight: bold;
        }

        .change-item.positive {
            background: #d4edda;
            color: #155724;
        }

        .change-item.negative {
            background: #f8d7da;
            color: #721c24;
        }

        .change-item.neutral {
            background: #e2e3e5;
            color: #383d41;
        }

        .trends {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
        }

        .trend-card {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
            border-left: 4px solid var(--info-color);
        }

        .trend-header {
            font-weight: bold;
            margin-bottom: 10px;
            color: var(--secondary-color);
        }

        .trend-value {
            font-size: 1.2em;
            font-weight: bold;
        }

        .trend-up {
            color: var(--danger-color);
        }

        .trend-down {
            color: var(--success-color);
        }

        .trend-stable {
            color: #666;
        }

        .snapshot-comparison {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 30px;
        }

        .snapshot {
            background: #f8f9fa;
            border-radius: 8px;
            padding: 20px;
        }

        .snapshot.current {
            border-left: 4px solid var(--success-color);
        }

        .snapshot.previous {
            border-left: 4px solid var(--info-color);
        }

        .snapshot-header {
            font-weight: bold;
            margin-bottom: 15px;
            color: var(--secondary-color);
        }

        .metric-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
            padding: 8px 0;
            border-bottom: 1px solid #eee;
        }

        .metric-row:last-child {
            border-bottom: none;
        }

        .metric-label {
            color: #666;
        }

        .metric-value {
            font-weight: bold;
            color: var(--secondary-color);
        }

        .no-data {
            text-align: center;
            color: #666;
            font-style: italic;
            padding: 40px;
        }

        @media (max-width: 768px) {
            .container {
                padding: 10px;
            }

            .stats-grid {
                grid-template-columns: 1fr;
            }

            .snapshot-comparison {
                grid-template-columns: 1fr;
            }

            .trends {
                grid-template-columns: 1fr;
            }

            .timeline-stats {
                grid-template-columns: 1fr 1fr;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>Configuration Timeline</h1>
            <div class="subtitle">{{ .ConfigName }}</div>
            <div class="subtitle">{{ .GeneratedAt }}</div>
        </div>

        <div class="stats-grid">
            <div class="stat-card">
                <span class="stat-number">{{ .TotalVersions }}</span>
                <div class="stat-label">Total Versions</div>
            </div>
            <div class="stat-card">
                <span class="stat-number">{{ .CurrentSnapshot.TotalResources }}</span>
                <div class="stat-label">Current Resources</div>
            </div>
            <div class="stat-card danger">
                <span class="stat-number">{{ .CurrentSnapshot.TotalSecurityIssues }}</span>
                <div class="stat-label">Security Issues</div>
            </div>
            <div class="stat-card">
                <span class="stat-number">{{ .TimeSpan }}</span>
                <div class="stat-label">Time Span</div>
            </div>
        </div>

        {{if .PreviousSnapshot}}
        <div class="section">
            <div class="section-header">Current vs Previous Snapshot</div>
            <div class="section-content">
                <div class="snapshot-comparison">
                    <div class="snapshot current">
                        <div class="snapshot-header">Current ({{ .CurrentSnapshot.FormattedTime }})</div>
                        <div class="metric-row">
                            <span class="metric-label">Total Resources</span>
                            <span class="metric-value">{{ .CurrentSnapshot.TotalResources }}</span>
                        </div>
                        <div class="metric-row">
                            <span class="metric-label">Privileged Containers</span>
                            <span class="metric-value">{{ .CurrentSnapshot.PrivilegedContainers }}</span>
                        </div>
                        <div class="metric-row">
                            <span class="metric-label">Containers w/ Capabilities</span>
                            <span class="metric-value">{{ .CurrentSnapshot.CapabilityContainers }}</span>
                        </div>
                        <div class="metric-row">
                            <span class="metric-label">Host Namespace Usage</span>
                            <span class="metric-value">{{ .CurrentSnapshot.HostNamespaceUsage }}</span>
                        </div>
                        <div class="metric-row">
                            <span class="metric-label">Host Path Volumes</span>
                            <span class="metric-value">{{ .CurrentSnapshot.HostPathVolumes }}</span>
                        </div>
                    </div>
                    <div class="snapshot previous">
                        <div class="snapshot-header">Previous ({{ .PreviousSnapshot.FormattedTime }})</div>
                        <div class="metric-row">
                            <span class="metric-label">Total Resources</span>
                            <span class="metric-value">{{ .PreviousSnapshot.TotalResources }}</span>
                        </div>
                        <div class="metric-row">
                            <span class="metric-label">Privileged Containers</span>
                            <span class="metric-value">{{ .PreviousSnapshot.PrivilegedContainers }}</span>
                        </div>
                        <div class="metric-row">
                            <span class="metric-label">Containers w/ Capabilities</span>
                            <span class="metric-value">{{ .PreviousSnapshot.CapabilityContainers }}</span>
                        </div>
                        <div class="metric-row">
                            <span class="metric-label">Host Namespace Usage</span>
                            <span class="metric-value">{{ .PreviousSnapshot.HostNamespaceUsage }}</span>
                        </div>
                        <div class="metric-row">
                            <span class="metric-label">Host Path Volumes</span>
                            <span class="metric-value">{{ .PreviousSnapshot.HostPathVolumes }}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        {{end}}

        {{if .ResourceTrends}}
        <div class="section">
            <div class="section-header">Resource Trends</div>
            <div class="section-content">
                <div class="trends">
                    {{range .ResourceTrends}}
                    <div class="trend-card">
                        <div class="trend-header">{{ .ResourceType }}</div>
                        <div class="trend-value {{ trendClass .Trend }}">
                            {{ formatChange .TotalChange }} ({{ .Trend }})
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
        </div>
        {{end}}

        {{if .SecurityTrends}}
        <div class="section">
            <div class="section-header">Security Trends</div>
            <div class="section-content">
                <div class="trends">
                    {{range .SecurityTrends}}
                    <div class="trend-card">
                        <div class="trend-header">{{ .FindingType }}</div>
                        <div class="trend-value {{ trendClass .Trend }}">
                            {{ formatChange .TotalChange }} ({{ .Trend }})
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
        </div>
        {{end}}

        <div class="section">
            <div class="section-header">Configuration Timeline</div>
            <div class="section-content">
                {{if .Timeline}}
                <div class="timeline">
                    {{range .Timeline}}
                    <div class="timeline-entry{{if .IsLatest}} latest{{end}}">
                        <div class="timeline-time">
                            {{ .FormattedTime }}{{if .IsLatest}} (Latest){{end}}
                        </div>
                        
                        <div class="timeline-stats">
                            <div class="timeline-stat">
                                <div class="timeline-stat-value">{{ .TotalResources }}</div>
                                <div class="timeline-stat-label">Resources</div>
                            </div>
                            <div class="timeline-stat">
                                <div class="timeline-stat-value">{{ .SecurityIssues }}</div>
                                <div class="timeline-stat-label">Security Issues</div>
                            </div>
                        </div>

                        {{if .ResourceChanges}}
                        <div class="changes">
                            <strong>Resource Changes:</strong>
                            {{range $type, $change := .ResourceChanges}}
                            <span class="change-item {{ changeClass $change }}">
                                {{ $type }}: {{ formatChange $change }}
                            </span>
                            {{end}}
                        </div>
                        {{end}}

                        {{if .SecurityChanges}}
                        <div class="changes">
                            <strong>Security Changes:</strong>
                            {{range $type, $change := .SecurityChanges}}
                            <span class="change-item {{ changeClass $change }}">
                                {{ $type }}: {{ formatChange $change }}
                            </span>
                            {{end}}
                        </div>
                        {{end}}
                    </div>
                    {{end}}
                </div>
                {{else}}
                <div class="no-data">No timeline data available</div>
                {{end}}
            </div>
        </div>
    </div>
</body>
</html>`