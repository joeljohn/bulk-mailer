package main

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"net/smtp"
	"os"
	"regexp"
	"text/template"
	"time"

	"github.com/urfave/cli"
)

func main() {

	var templateFileName, recipientListFileName, configFileName string

	//For reading the template and recipientList filepaths, cli is utilized
	// https://github.com/urfave/cli
	app := &cli.App{
		Name:  "bmail",
		Usage: "Send Bulk Emails",
		//cli flags take Filepath from user
		Flags: []cli.Flag{
			//cli flag for taking TemplateFilepath from user
			&cli.StringFlag{
				Name:     "template, t",
				Usage:    "Load HTML template from `FILE`",
				Required: true,
			},
			//cli flag for taking RecipientlistFilepath from user
			&cli.StringFlag{
				Name:     "recipient, r",
				Usage:    "Load recipient list (csv) from `FILE`",
				Required: true,
			},
			//cli flag for taking SMTPConfigFilepath from user
			&cli.StringFlag{
				Name:     "config, c",
				Usage:    "Load SMTPConfig File (csv) from `FILE`",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {

			templateFileName = c.String("template")
			recipientListFileName = c.String("recipient")
			configFileName = c.String("config")
			fmt.Println("Use bmail --template TEMPLATEFILE.html --recipient RECIPIENTLIST.csv")
			return nil
		},
	}
	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
	subject := "Test Mail"
	ReadRecipient(recipientListFileName, templateFileName, configFileName, subject)

}

//Message struct
type Message struct {
	to      string
	from    string
	subject string
	body    string
}

//ParseTemplate parses the template
func ParseTemplate(templateFileName string, data interface{}) string {

	// Open the file
	tmpl, err := template.ParseFiles(templateFileName)
	if err != nil {
		log.Fatalln("Couldn't open the template", err)
	}
	buf := new(bytes.Buffer)
	//data is inserted into {{.}} fields
	tmpl.Execute(buf, data)
	return buf.String()
}

//ReadRecipient reads list of recipients from csv file
func ReadRecipient(recipientListFileName, templateFileName, configFileName, subject string) {

	configFile, err := os.Open(configFileName)
	if err != nil {
		log.Fatalln("Couldn't open the csv file", err)
	}
	// Parse the file
	r := csv.NewReader(bufio.NewReader(configFile))

	var username, password, hostname, port string

	for {
		// Read each smtp records details from csv
		serverRecord, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		username = serverRecord[0]
		password = serverRecord[1]
		hostname = serverRecord[2]
		port = serverRecord[3]
	}

	// Open the recipient list
	csvFile, err := os.Open(recipientListFileName)
	if err != nil {
		log.Fatalln("Couldn't open the csv file", err)
	}
	// Parse the file
	reader := csv.NewReader(bufio.NewReader(csvFile))

	// Iterate through the records
	for {
		// Read each record from csv
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}

		//validating email structure using regex
		erre := ValidateFormat(record[1])
		if erre != nil {
			fmt.Println("Email address (", record[1], ") is not valid - Skipping...")
		}
		if erre == nil {
			//Structure for sending data to
			data := struct {
				Name string
			}{
				Name: record[0],
			}
			//Parsing data to template (i.e "Name" in place of {{.Name}})
			body := ParseTemplate(templateFileName, data)
			m := Message{
				to:      record[1],
				subject: subject,
				body:    body,
				from:    "mail@example.com",
			}
			m.Send(username, password, hostname, port)

		}
	}
}

//Send for sending email
func (m *Message) Send(username, password, hostname, port string) {
	// Set up authentication information.
	//i, _ := strconv.Atoi(port)
	auth := smtp.PlainAuth("", username, password, hostname)
	addr := hostname + ":" + port

	//Convert "to" to []string
	to := []string{m.to}
	//RFC 822-style email format
	//Omit "to" parameter in msg to send as bcc
	msg := []byte("From: " + m.from + "\r\n" +
		"To: " + m.to + "\r\n" +
		"Subject: " + m.subject + "\r\n" +
		"MIME-version: 1.0;\nContent-Type: text/html; charset=\"UTF-8\";\n\n" +
		"\r\n" +
		m.body + "\r\n")

	err := smtp.SendMail(addr, auth, username, to, msg)
	//If an error occurs while sending emails, it will try 10 times(waiting for 2 seconds each time)
	count := 0
	for err != nil && count <= 10 {
		time.Sleep(2 * time.Second)
		err = smtp.SendMail(addr, auth, username, to, msg)
		count++
	}
	if err != nil {
		fmt.Println(err)
		fmt.Println("Failed sending to ", m.to)
	} else {
		fmt.Println("Email Sent to ", m.to)
	}

}

// ValidateFormat validates the email using regex
func ValidateFormat(email string) error {
	regex := regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")
	if !regex.MatchString(email) {
		return errors.New("Invalid Format")
	}
	return nil
}
