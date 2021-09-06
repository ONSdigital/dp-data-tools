package main

import (
	"encoding/csv"
	"fmt"
	"strings"

	"os"

	"time"

	"github.com/ONSdigital/dp-zebedee-sdk-go/zebedee"
	"github.com/google/uuid"
)

type config struct {
	filename,
	host,
	pword,
	user string
}

var header = cognito_user{
	username:              "cognito:username",
	name:                  "name",
	given_name:            "given_name",
	family_name:           "family_name",
	middle_name:           "middle_name",
	nickname:              "nickname",
	preferred_username:    "preferred_username",
	profile:               "profile",
	picture:               "picture",
	website:               "website",
	email:                 "email",
	email_verified:        "email_verified",
	gender:                "gender",
	birthdate:             "birthdate",
	zoneinfo:              "zoneinfo",
	locale:                "locale",
	phone_number:          "phone_number",
	phone_number_verified: "phone_number_verified",
	address:               "address",
	updated_at:            "updated_at",
	mfa_enabled:           "cognito:mfa_enabled",
}

type cognito_user struct {
	username,
	name,
	given_name,
	family_name,
	middle_name,
	nickname,
	preferred_username,
	profile,
	picture,
	website,
	email,
	email_verified,
	gender,
	birthdate,
	zoneinfo,
	locale,
	phone_number,
	phone_number_verified,
	address,
	updated_at,
	mfa_enabled string
}

func readConfig() *config {
	conf := &config{}
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if pair[0] == "filename" {
			conf.filename = pair[1]
		}
		if pair[0] == "zebedee_user" {
			conf.user = pair[1]
		}
		if pair[0] == "zebedee_pword" {
			conf.pword = pair[1]
		}
		if pair[0] == "zebedee_host" {
			conf.host = pair[1]
		}
	}
	if conf.host == "" || conf.pword == "" || conf.user == "" || conf.filename == "" {
		fmt.Println("Please set Environment Variables ")
		os.Exit(1)
	}

	return conf
}

func convert_to_slice(input cognito_user) []string {
	return []string{
		input.username,
		input.name,
		input.given_name,
		input.family_name,
		input.middle_name,
		input.nickname,
		input.preferred_username,
		input.profile,
		input.picture,
		input.website,
		input.email,
		input.email_verified,
		input.gender,
		input.birthdate,
		input.zoneinfo,
		input.locale,
		input.phone_number,
		input.phone_number_verified,
		input.address,
		input.updated_at,
		input.mfa_enabled,
	}
}

func process_zebedee_users(csvwriter *csv.Writer, userlist []zebedee.User) {
	for _, user := range userlist {
		var (
			csvline cognito_user
		)
		csvline.username = uuid.NewString()
		csvline.email = user.Email

		domain := strings.Split(user.Email, "@")
		names := strings.Split(domain[0], ".")

		if len(names) == 2 {
			csvline.given_name = names[0]
			csvline.family_name = names[1]
		} else if len(names) > 2 {
			csvline.given_name = names[0]
			csvline.family_name = names[2]
		} else {
			csvline.given_name = ""
			csvline.family_name = names[0]
		}

		csvline.mfa_enabled = "FALSE"
		csvline.phone_number_verified = "FALSE"
		csvline.email_verified = "TRUE"
		if err := csvwriter.Write(convert_to_slice(csvline)); err != nil {
			fmt.Println("error writing record to csv:", err)
		}
	}
}

func main() {

	conf := readConfig()
	httpCli := zebedee.NewHttpClient(time.Second * 5)
	zebCli := zebedee.NewClient(conf.host, httpCli)

	c := zebedee.Credentials{
		Email:    conf.user,
		Password: conf.pword,
	}

	sess, err := zebCli.OpenSession(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	userList, err := zebCli.GetUsers(sess)

	if err != nil {
		fmt.Println("Theres been an issue")
		fmt.Println(err)
		os.Exit(1)
	}

	csvfile, err := os.Create(conf.filename)
	if err != nil {
		fmt.Printf("failed creating file: %s", err)
		os.Exit(1)
	}
	csvwriter := csv.NewWriter(csvfile)

	csvheader := convert_to_slice(header)
	csvwriter.Write(csvheader)

	process_zebedee_users(csvwriter, userList)

	csvwriter.Flush()

	fmt.Println("There are ", len(userList), "records extracted to file", conf.filename, "csv Errors ", csvwriter.Error())
	csvfile.Close()

}
