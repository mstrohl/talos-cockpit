package main

import (
	"bytes"
	"log"
	"text/template"

	"gopkg.in/gomail.v2"
)

type NodeUpdateReport struct {
	NodeName          string
	PreviousVersion   string
	ImageSource       string
	NewVersion        string
	UpdateStatus      string
	StatusClass       string
	AdditionalDetails string
	Timestamp         string
	ClusterID         string
}

type K8SUpdateReport struct {
	ClusterID         string
	UpdateStatus      string
	StatusClass       string
	AdditionalDetails string
	Timestamp         string
}

// SendMail for Alerting purpose
func sendMail(subject string, body string) {

	var cfg Config
	var MailRecipient string
	//var MailCc string
	var MailHost string
	var MailUsername string
	var MailPassword string
	readFile(&cfg)
	readEnv(&cfg)
	//fmt.Printf("%+v\n", cfg)

	if cfg.Notifications.Mail.Host != "" && cfg.Notifications.Mail.User != "" && cfg.Notifications.Mail.Password != "" && cfg.Notifications.Mail.Recipient != "" {
		MailRecipient = cfg.Notifications.Mail.Recipient
		//MailCc = cfg.Notifications.Mail.Cc
		MailHost = cfg.Notifications.Mail.Host
		MailUsername = cfg.Notifications.Mail.User
		MailPassword = cfg.Notifications.Mail.Password
	} else {
		log.Printf("WARNING - Notification Mail configuration missing")
		return
	}

	s := gomail.NewMessage()
	s.SetHeader("From", "noreply@dive-in-it.com")
	s.SetHeader("To", MailRecipient)
	//m.SetHeader("To", "recipient1@example.com", "recipient2@example.com")
	//s.SetAddressHeader("Cc", MailCc, "CC Name")
	//m.SetAddressHeader("Bcc", "blindcarboncopy@example.com", "BCC Name")
	s.SetHeader("Subject", subject)
	s.SetBody("text/html", body)
	d := gomail.NewDialer(MailHost, 587, MailUsername, MailPassword)
	if err := d.DialAndSend(s); err != nil {
		panic(err)
	}
}

// Identifying status to change color if needed
func determineStatusClass(status string) string {
	switch status {
	case "Success":
		return "status-ok"
	case "Partial":
		return "status-warning"
	case "Failed":
		return "status-error"
	default:
		return "status-warning"
	}
}

// Gen email body HTML
func generateUpdateEmailBody(report NodeUpdateReport) (string, error) {
	// HTML template as a string
	htmlTemplate := `<!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <style>
            body {
                font-family: Arial, sans-serif;
                line-height: 1.6;
                color: #333;
                max-width: 600px;
                margin: 0 auto;
                padding: 20px;
            }
            .header {
                background-color: #f4f4f4;
                padding: 10px;
                text-align: center;
                border-bottom: 2px solid #007bff;
            }
            .content {
                padding: 20px;
            }
            .node-info {
                background-color: #e9ecef;
                border-radius: 5px;
                padding: 15px;
                margin-bottom: 15px;
            }
            .status-ok {
                color: #28a745;
                font-weight: bold;
            }
            .status-warning {
                color: #ffc107;
                font-weight: bold;
            }
            .status-error {
                color: #dc3545;
                font-weight: bold;
            }
        </style>
    </head>
    <body>
        <div class="header">
            <h1>Talos Cockpit - Node Automatic Update Report</h1>
        </div>
        
        <div class="content">
            <div class="node-info">
                <h2>Node Details</h2>
                <p><strong>Node Name:</strong> {{.NodeName}}</p>
				<p><strong>Previous Version:</strong> {{.PreviousVersion}}</p>
                <p><strong>Image Source:</strong> {{.ImageSource}}</p>
                <p><strong>New Version:</strong> {{.NewVersion}}</p>
                <p><strong>Update Status:</strong> 
                    <span class="{{.StatusClass}}">{{.UpdateStatus}}</span>
                </p>
            </div>
            
            <div class="node-info">
                <h3>Update Details</h3>
                <p>{{.AdditionalDetails}}</p>
            </div>
            
            <p>This update was processed by Talos Cockpit on {{.Timestamp}}</p>
        </div>
    </body>
    </html>`

	// Create a new template and parse the HTML
	tmpl, err := template.New("emailTemplate").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	// Buffer to store the rendered template
	var renderedTemplate bytes.Buffer

	// Execute the template with the report data
	err = tmpl.Execute(&renderedTemplate, report)
	if err != nil {
		return "", err
	}

	return renderedTemplate.String(), nil
}

func generateK8SUpdateEmailBody(report K8SUpdateReport) (string, error) {
	// HTML template as a string
	htmlTemplate := `<!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <style>
            body {
                font-family: Arial, sans-serif;
                line-height: 1.6;
                color: #333;
                max-width: 600px;
                margin: 0 auto;
                padding: 20px;
            }
            .header {
                background-color: #f4f4f4;
                padding: 10px;
                text-align: center;
                border-bottom: 2px solid #007bff;
            }
            .content {
                padding: 20px;
            }
            .node-info {
                background-color: #e9ecef;
                border-radius: 5px;
                padding: 15px;
                margin-bottom: 15px;
            }
            .status-ok {
                color: #28a745;
                font-weight: bold;
            }
            .status-warning {
                color: #ffc107;
                font-weight: bold;
            }
            .status-error {
                color: #dc3545;
                font-weight: bold;
            }
        </style>
    </head>
    <body>
        <div class="header">
            <h1>Talos Cockpit - Kubernetes Update Report</h1>
        </div>
        
        <div class="content">
            <div class="node-info">
                <h2>Node Details</h2>
                <p><strong>ClusterID:</strong> {{.ClusterID}}</p>
                <p><strong>Update Status:</strong> 
                    <span class="{{.StatusClass}}">{{.UpdateStatus}}</span>
                </p>
            </div>
            
            <div class="node-info">
                <h3>Update Details</h3>
                <p>{{.AdditionalDetails}}</p>
            </div>
            
            <p>This update was processed by Talos Cockpit on {{.Timestamp}}</p>
        </div>
    </body>
    </html>`

	// Create a new template and parse the HTML
	tmpl, err := template.New("emailTemplate").Parse(htmlTemplate)
	if err != nil {
		return "", err
	}

	// Buffer to store the rendered template
	var renderedTemplate bytes.Buffer

	// Execute the template with the report data
	err = tmpl.Execute(&renderedTemplate, report)
	if err != nil {
		return "", err
	}

	return renderedTemplate.String(), nil
}
