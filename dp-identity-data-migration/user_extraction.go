package main

import (
	"encoding/csv"
	"fmt"
	"strings"

	// "io/ioutil"
	"os"

	"time"

	"github.com/ONSdigital/dp-zebedee-sdk-go/zebedee"
	"github.com/google/uuid"
)

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

func populate_Header() (header cognito_user) {
	header = cognito_user{
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
	return header
}
func convert_cognito_user_to_slice(input cognito_user) (output []string) {
	output = []string{
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
	return output
}

func getUsers(zebCli zebedee.Client, s zebedee.Session) (userlist []zebedee.User, err error) {

	userList, err := zebCli.GetUsers(s)

	if err != nil {
		fmt.Println("get users error!")
		return nil, err
	}

	for _, user := range userList {
		fmt.Println(user)
	}
	return userList, nil
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
		if err := csvwriter.Write(convert_cognito_user_to_slice(csvline)); err != nil {
			fmt.Println("error writing record to csv:", err)
		}
	}
}

func readConfig() (filename, host, pword, user string) {
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if pair[0] == "filename" {
			filename = pair[1]
		}
		if pair[0] == "zebedee_user" {
			user = pair[1]
		}
		if pair[0] == "zebedee_pword" {
			pword = pair[1]
		}
		if pair[0] == "zebedee_host" {
			host = pair[1]
		}
	}
	if host == "" || pword == "" || user == "" || filename == "" {
		fmt.Println("Please set Environment Variables ")
		os.Exit(1)
	}

	return filename, host, pword, user
}

func main() {

	filename, host, pword, user := readConfig()

	httpCli := zebedee.NewHttpClient(time.Second * 5)
	zebCli := zebedee.NewClient(host, httpCli)

	c := zebedee.Credentials{
		Email:    user,
		Password: pword,
	}

	sess, err := zebCli.OpenSession(c)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	userList, err := getUsers(zebCli, sess)

	if err != nil {
		fmt.Println("Theres been an issue")
		fmt.Println(err)
		os.Exit(1)
	}

	csvfile, err := os.Create(filename)
	if err != nil {
		fmt.Printf("failed creating file: %s", err)
		os.Exit(1)
	}
	csvwriter := csv.NewWriter(csvfile)

	header := populate_Header()
	csvheader := convert_cognito_user_to_slice(header)
	csvwriter.Write(csvheader)

	process_zebedee_users(csvwriter, userList)

	csvwriter.Flush()

	fmt.Println(len(userList), csvwriter.Error())
	csvfile.Close()

}
