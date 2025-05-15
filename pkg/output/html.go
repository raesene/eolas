package output

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"time"

	"github.com/raesene/eolas/pkg/kubernetes"
)

// HTMLFormatter generates HTML output for analysis results
type HTMLFormatter struct {
	template *template.Template
}

// HTMLData represents the data passed to the HTML template
type HTMLData struct {
	Title             string
	GeneratedAt       string
	ClusterName       string
	ResourceCounts    map[string]int
	TotalResources    int
	PrivilegedResults []kubernetes.PrivilegedContainer
	CapabilityResults []kubernetes.CapabilityContainer
	HostNSResults     []kubernetes.HostNamespaceWorkload
	HostPathResults   []kubernetes.HostPathVolume
}

// NewHTMLFormatter creates a new HTML formatter with the embedded template
func NewHTMLFormatter() (*HTMLFormatter, error) {
	tmpl, err := template.New("html").Parse(htmlTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML template: %w", err)
	}

	return &HTMLFormatter{
		template: tmpl,
	}, nil
}

// GenerateHTML creates HTML content from analysis results
func (f *HTMLFormatter) GenerateHTML(
	clusterName string,
	resourceCounts map[string]int,
	privilegedResults []kubernetes.PrivilegedContainer,
	capabilityResults []kubernetes.CapabilityContainer,
	hostNSResults []kubernetes.HostNamespaceWorkload,
	hostPathResults []kubernetes.HostPathVolume,
) ([]byte, error) {
	// Calculate total resources
	totalResources := 0
	for _, count := range resourceCounts {
		totalResources += count
	}

	// Prepare data for template
	data := HTMLData{
		Title:             "Eolas Kubernetes Analysis Report",
		GeneratedAt:       time.Now().Format(time.RFC1123),
		ClusterName:       clusterName,
		ResourceCounts:    resourceCounts,
		TotalResources:    totalResources,
		PrivilegedResults: privilegedResults,
		CapabilityResults: capabilityResults,
		HostNSResults:     hostNSResults,
		HostPathResults:   hostPathResults,
	}

	var buf bytes.Buffer
	if err := f.template.Execute(&buf, data); err != nil {
		return nil, fmt.Errorf("failed to execute HTML template: %w", err)
	}

	return buf.Bytes(), nil
}

// WriteHTMLToFile writes HTML content to a file
func (f *HTMLFormatter) WriteHTMLToFile(content []byte, filePath string) error {
	return os.WriteFile(filePath, content, 0644)
}

// htmlTemplate is the embedded HTML template with CSS and JavaScript
const htmlTemplate = `<!DOCTYPE html>
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
            --table-header-bg: #f2f2f2;
            --warning-color: #e74c3c;
            --success-color: #2ecc71;
            --info-color: #3498db;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            line-height: 1.6;
            color: var(--text-color);
            background-color: var(--background-color);
            margin: 0;
            padding: 20px;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            box-shadow: 0 0 10px rgba(0, 0, 0, 0.1);
            border-radius: 8px;
            padding: 20px;
        }

        h1, h2, h3 {
            color: var(--secondary-color);
        }

        h1 {
            text-align: center;
            margin-bottom: 30px;
            padding-bottom: 15px;
            border-bottom: 2px solid var(--primary-color);
        }

        h2 {
            margin-top: 30px;
            padding-bottom: 10px;
            border-bottom: 1px solid var(--border-color);
        }

        table {
            width: 100%;
            border-collapse: collapse;
            margin: 20px 0;
            box-shadow: 0 2px 5px rgba(0, 0, 0, 0.1);
        }

        th, td {
            padding: 12px 15px;
            border: 1px solid var(--border-color);
            text-align: left;
        }

        th {
            background-color: var(--table-header-bg);
            font-weight: bold;
        }

        tr:nth-child(even) {
            background-color: rgba(0, 0, 0, 0.02);
        }

        tr:hover {
            background-color: rgba(0, 0, 0, 0.05);
        }

        .badge {
            display: inline-block;
            padding: 4px 8px;
            border-radius: 4px;
            color: white;
            font-size: 0.85em;
            margin-right: 5px;
        }

        .badge-true {
            background-color: var(--warning-color);
        }

        .badge-false {
            background-color: var(--success-color);
        }

        .report-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 20px;
        }

        .report-meta {
            background-color: var(--secondary-color);
            color: white;
            padding: 10px;
            border-radius: 5px;
            margin-bottom: 20px;
        }

        .report-meta p {
            margin: 5px 0;
        }

        .section {
            margin-bottom: 30px;
            padding-bottom: 20px;
            border-bottom: 1px dashed var(--border-color);
        }

        .section:last-child {
            border-bottom: none;
        }

        .warning-text {
            color: var(--warning-color);
            font-weight: bold;
        }

        .note {
            background-color: #f8f9fa;
            border-left: 4px solid var(--info-color);
            padding: 15px;
            margin: 20px 0;
            border-radius: 4px;
        }

        .tabs {
            display: flex;
            border-bottom: 1px solid var(--border-color);
            margin-bottom: 20px;
        }

        .tab {
            padding: 10px 20px;
            cursor: pointer;
            margin-right: 5px;
            border: 1px solid transparent;
            border-bottom: none;
            border-radius: 5px 5px 0 0;
        }

        .tab.active {
            border-color: var(--border-color);
            background-color: white;
            margin-bottom: -1px;
        }

        .tab-content {
            display: none;
        }

        .tab-content.active {
            display: block;
        }

        .footer {
            text-align: center;
            margin-top: 30px;
            padding-top: 20px;
            border-top: 1px solid var(--border-color);
            color: #777;
            font-size: 0.9em;
        }
        
        .summary-box {
            background-color: #f1f8ff;
            border: 1px solid var(--primary-color);
            border-radius: 8px;
            padding: 15px;
            margin: 20px 0;
        }
        
        .summary-box h3 {
            margin-top: 0;
            color: var(--primary-color);
        }
        
        .resource-counts {
            display: flex;
            flex-wrap: wrap;
            gap: 10px;
        }
        
        .resource-count-item {
            background-color: #fff;
            border: 1px solid var(--border-color);
            border-radius: 6px;
            padding: 10px;
            min-width: 150px;
            box-shadow: 0 1px 3px rgba(0,0,0,0.05);
        }
        
        .resource-count-item .name {
            font-weight: bold;
        }
        
        .resource-count-item .count {
            font-size: 1.2em;
            color: var(--primary-color);
        }
        
        .alert {
            padding: 15px;
            margin: 20px 0;
            border-radius: 6px;
            border-left: 5px solid;
        }
        
        .alert-warning {
            background-color: #fff3cd;
            border-left-color: #ffc107;
        }
        
        .alert-danger {
            background-color: #f8d7da;
            border-left-color: #dc3545;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>{{ .Title }}</h1>
        
        <div class="report-meta">
            <p><strong>Cluster:</strong> {{ .ClusterName }}</p>
            <p><strong>Generated at:</strong> {{ .GeneratedAt }}</p>
            <p><strong>Total Resources:</strong> {{ .TotalResources }}</p>
        </div>

        <div class="tabs">
            <div class="tab active" onclick="showTab('overview')">Overview</div>
            <div class="tab" onclick="showTab('privileged')">Privileged Containers</div>
            <div class="tab" onclick="showTab('capabilities')">Linux Capabilities</div>
            <div class="tab" onclick="showTab('host-namespaces')">Host Namespaces</div>
            <div class="tab" onclick="showTab('host-paths')">Host Path Volumes</div>
        </div>

        <!-- Overview Tab Content -->
        <div id="overview" class="tab-content active">
            <h2>Overview</h2>
            
            <div class="summary-box">
                <h3>Security Findings Summary</h3>
                <ul>
                    <li><strong>Privileged Containers:</strong> {{ len .PrivilegedResults }}</li>
                    <li><strong>Containers with Added Capabilities:</strong> {{ len .CapabilityResults }}</li>
                    <li><strong>Workloads Using Host Namespaces:</strong> {{ len .HostNSResults }}</li>
                </ul>
            </div>
            
            <h3>Resource Counts</h3>
            
            <div class="resource-counts">
                {{ range $key, $value := .ResourceCounts }}
                <div class="resource-count-item">
                    <div class="name">{{ $key }}</div>
                    <div class="count">{{ $value }}</div>
                </div>
                {{ end }}
            </div>
        </div>

        <!-- Privileged Containers Tab Content -->
        <div id="privileged" class="tab-content">
            <h2>Privileged Containers</h2>
            
            {{ if .PrivilegedResults }}
            <div class="alert alert-danger">
                <p><strong>Warning:</strong> Found {{ len .PrivilegedResults }} privileged containers in the cluster.</p>
                <p>Privileged containers have full access to the host's kernel capabilities and device nodes, similar to root access on the host. These should be reviewed carefully for security implications.</p>
            </div>
            
            <table>
                <thead>
                    <tr>
                        <th>Namespace</th>
                        <th>Resource Type</th>
                        <th>Resource Name</th>
                        <th>Container Name</th>
                    </tr>
                </thead>
                <tbody>
                    {{ range .PrivilegedResults }}
                    <tr>
                        <td>{{ if .Namespace }}{{ .Namespace }}{{ else }}default{{ end }}</td>
                        <td>{{ .Kind }}</td>
                        <td>{{ .PodName }}</td>
                        <td>{{ .Name }}</td>
                    </tr>
                    {{ end }}
                </tbody>
            </table>
            {{ else }}
            <p>No privileged containers found in the cluster. 👍</p>
            {{ end }}
        </div>

        <!-- Capabilities Tab Content -->
        <div id="capabilities" class="tab-content">
            <h2>Containers with Added Linux Capabilities</h2>
            
            {{ if .CapabilityResults }}
            <div class="alert alert-warning">
                <p><strong>Caution:</strong> Found {{ len .CapabilityResults }} containers with added Linux capabilities.</p>
                <p>Added Linux capabilities provide containers with elevated privileges. Particularly dangerous capabilities include: CAP_SYS_ADMIN, CAP_NET_ADMIN, CAP_SYS_PTRACE, and CAP_NET_RAW. These should be reviewed for necessity.</p>
            </div>
            
            <table>
                <thead>
                    <tr>
                        <th>Namespace</th>
                        <th>Resource Type</th>
                        <th>Resource Name</th>
                        <th>Container</th>
                        <th>Capabilities</th>
                    </tr>
                </thead>
                <tbody>
                    {{ range .CapabilityResults }}
                    <tr>
                        <td>{{ if .Namespace }}{{ .Namespace }}{{ else }}default{{ end }}</td>
                        <td>{{ .Kind }}</td>
                        <td>{{ .PodName }}</td>
                        <td>{{ .Name }}</td>
                        <td>
                            {{ range .Capabilities }}
                            <span class="badge {{ if or (eq . "CAP_SYS_ADMIN") (eq . "CAP_NET_ADMIN") (eq . "CAP_SYS_PTRACE") (eq . "CAP_NET_RAW") (eq . "NET_ADMIN") (eq . "SYS_ADMIN") (eq . "SYS_PTRACE") (eq . "NET_RAW") }}badge-true{{ else }}badge-false{{ end }}">{{ . }}</span>
                            {{ end }}
                        </td>
                    </tr>
                    {{ end }}
                </tbody>
            </table>
            {{ else }}
            <p>No containers with added Linux capabilities found in the cluster. 👍</p>
            {{ end }}
        </div>

        <!-- Host Namespaces Tab Content -->
        <div id="host-namespaces" class="tab-content">
            <h2>Workloads Using Host Namespaces</h2>
            
            {{ if .HostNSResults }}
            <div class="alert alert-danger">
                <p><strong>Warning:</strong> Found {{ len .HostNSResults }} workloads using host namespaces.</p>
                <p>Host namespaces provide containers with access to the host's resources. These pose significant security risks because they reduce isolation between containers and the host system.</p>
            </div>
            
            <table>
                <thead>
                    <tr>
                        <th>Namespace</th>
                        <th>Resource Type</th>
                        <th>Name</th>
                        <th>Host PID</th>
                        <th>Host IPC</th>
                        <th>Host Network</th>
                        <th>Host Ports</th>
                    </tr>
                </thead>
                <tbody>
                    {{ range .HostNSResults }}
                    <tr>
                        <td>{{ if .Namespace }}{{ .Namespace }}{{ else }}default{{ end }}</td>
                        <td>{{ .Kind }}</td>
                        <td>{{ .Name }}</td>
                        <td><span class="badge {{ if .HostPID }}badge-true{{ else }}badge-false{{ end }}">{{ .HostPID }}</span></td>
                        <td><span class="badge {{ if .HostIPC }}badge-true{{ else }}badge-false{{ end }}">{{ .HostIPC }}</span></td>
                        <td><span class="badge {{ if .HostNetwork }}badge-true{{ else }}badge-false{{ end }}">{{ .HostNetwork }}</span></td>
                        <td>
                            {{ if .HostPorts }}
                                {{ range .HostPorts }}{{ . }} {{ end }}
                            {{ else }}
                                None
                            {{ end }}
                        </td>
                    </tr>
                    {{ end }}
                </tbody>
            </table>
            
            <div class="note">
                <p><strong>Security Implications:</strong></p>
                <ul>
                    <li><strong>hostPID:</strong> Allows visibility of all processes on the host system</li>
                    <li><strong>hostIPC:</strong> Enables shared memory access with the host and all containers</li>
                    <li><strong>hostNetwork:</strong> Provides direct access to the host's network interfaces</li>
                    <li><strong>hostPorts:</strong> Exposes ports directly on the host's network interfaces</li>
                </ul>
            </div>
            {{ else }}
            <p>No workloads using host namespaces found in the cluster. 👍</p>
            {{ end }}
        </div>

        <!-- Host Path Volumes Tab Content -->
        <div id="host-paths" class="tab-content">
            <h2>Workloads Using Host Path Volumes</h2>
            
            {{ if .HostPathResults }}
            <div class="alert alert-danger">
                <p><strong>Warning:</strong> Found {{ len .HostPathResults }} workloads using hostPath volumes.</p>
                <p>hostPath volumes allow pods to access the host filesystem directly, potentially exposing sensitive files or enabling privilege escalation.</p>
            </div>
            
            <table class="data-table">
                <thead>
                    <tr>
                        <th>Namespace</th>
                        <th>Resource Type</th>
                        <th>Name</th>
                        <th>Host Path</th>
                        <th>Read-Only</th>
                    </tr>
                </thead>
                <tbody>
                    {{ range .HostPathResults }}
                        {{ $namespace := .Namespace }}
                        {{ $kind := .Kind }}
                        {{ $name := .Name }}
                        {{ $volume := . }}
                        {{ range $i, $path := .HostPaths }}
                            <tr>
                                {{ if eq $i 0 }}
                                <td>{{ if eq $namespace "" }}default{{ else }}{{ $namespace }}{{ end }}</td>
                                <td>{{ $kind }}</td>
                                <td>{{ $name }}</td>
                                {{ else }}
                                <td></td>
                                <td></td>
                                <td></td>
                                {{ end }}
                                <td>{{ $path }}</td>
                                <td>
                                    {{ if and (ge $i 0) (lt $i (len $volume.ReadOnly)) }}
                                        {{ if index $volume.ReadOnly $i }}Yes{{ else }}No{{ end }}
                                    {{ else }}
                                        Unknown
                                    {{ end }}
                                </td>
                            </tr>
                        {{ end }}
                    {{ end }}
                </tbody>
            </table>
            
            <div class="note">
                <h3>Security Implications</h3>
                <p>hostPath volumes pose significant security risks as they enable containers to access the host filesystem directly:</p>
                <ul>
                    <li><strong>Host System Access:</strong> Containers can read sensitive files from the host</li>
                    <li><strong>Data Persistence:</strong> Data can persist across pod restarts and be accessed by other pods</li>
                    <li><strong>Privilege Escalation:</strong> Write access to the host filesystem can enable privilege escalation</li>
                    <li><strong>Host Modification:</strong> Non read-only mounts allow containers to modify host files</li>
                </ul>
                <p>For improved security, consider using more restrictive volume types like emptyDir, configMap, or persistent volumes with appropriate access controls.</p>
            </div>
            {{ else }}
            <div class="alert alert-success">
                <p><strong>Good news!</strong> No workloads using hostPath volumes were found in the cluster.</p>
            </div>
            {{ end }}
        </div>

        <div class="footer">
            <p>Generated by Eolas - Kubernetes Cluster Analyzer</p>
            <p><a href="https://github.com/raesene/eolas" target="_blank">GitHub Repository</a></p>
        </div>
    </div>

    <script>
        function showTab(tabId) {
            // Hide all tab contents
            const tabContents = document.querySelectorAll('.tab-content');
            tabContents.forEach(content => {
                content.classList.remove('active');
            });
            
            // Remove active class from all tabs
            const tabs = document.querySelectorAll('.tab');
            tabs.forEach(tab => {
                tab.classList.remove('active');
            });
            
            // Show the selected tab content
            document.getElementById(tabId).classList.add('active');
            
            // Add active class to clicked tab
            event.currentTarget.classList.add('active');
        }
    </script>
</body>
</html>`