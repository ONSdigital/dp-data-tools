package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"

	"time"

	"github.com/ONSdigital/dp-zebedee-sdk-go/zebedee"
)

type config struct {
	groups_filename,
	groupusers_filename,
	host,
	pword,
	user string
}

func readConfig() (conf config) {
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		if pair[0] == "groups_filename" {
			conf.groups_filename = pair[1]
		}

		if pair[0] == "groupusers_filename" {
			conf.groupusers_filename = pair[1]
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
	if conf.host == "" || conf.pword == "" || conf.user == "" || conf.groups_filename == "" || conf.groupusers_filename == "" {
		fmt.Println("Please set Environment Variables ")
		os.Exit(1)
	}

	return conf
}

type group struct {
	GroupName,
	UserPoolId,
	Description,
	RoleArn,
	Precedence,
	LastModifiedDate,
	CreationDate string
}

var group_header = group{
	GroupName:        "group_name",
	UserPoolId:       "user_pool_id",
	Description:      "description",
	RoleArn:          "role_arn",
	Precedence:       "precedence",
	LastModifiedDate: "last_modified_date",
	CreationDate:     "creation_date",
}

type user_group struct {
	UserPoolId,
	Username,
	GroupName string
}

var user_group_header = user_group{
	UserPoolId: "user_pool_id",
	Username:   "user_name",
	GroupName:  "group_name",
}

func getgroups(zebCli zebedee.Client, s zebedee.Session) (grouplist zebedee.TeamsList, err error) {

	grouplist, err = zebCli.ListTeams(s)

	if err != nil {
		fmt.Println("get users error!")
		return grouplist, err
	}

	return grouplist, nil
}

func convert_to_slice_group(input group) []string {
	return []string{
		input.GroupName,
		input.UserPoolId,
		input.Description,
		input.RoleArn,
		input.Precedence,
		input.LastModifiedDate,
		input.CreationDate,
	}
}

func convert_to_slice_group_user(input user_group) []string {
	return []string{
		input.UserPoolId,
		input.Username,
		input.GroupName,
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

	groupList, err := getgroups(zebCli, sess)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	// from the groupList from zebedee
	// open csv file write group header and then for every line write line to csv file
	// flush and close csv

	groups_csvfile, err := os.Create(conf.groups_filename)
	if err != nil {
		fmt.Printf("failed creating file: %s", err)
		os.Exit(1)
	}
	csvwriter := csv.NewWriter(groups_csvfile)
	csvwriter.Write(convert_to_slice_group(group_header))
	for _, zebedeegroup := range groupList.Teams {
		tmp := group{
			GroupName:        zebedeegroup.Name,
			UserPoolId:       "",
			Description:      zebedeegroup.Name,
			RoleArn:          "",
			Precedence:       "10",
			LastModifiedDate: "",
			CreationDate:     "",
		}
		csvwriter.Write(convert_to_slice_group(tmp))

	}
	csvwriter.Flush()

	fmt.Println("There are ", len(groupList.Teams), "records extracted to file", conf.groups_filename, "csv Errors ", csvwriter.Error())
	groups_csvfile.Close()

	// from the groupList from zebedee
	// open csv file write usergroup header and then write csv line for every member in each line
	// flush and close csv

	usergroups_csvfile, err := os.Create(conf.groupusers_filename)
	if err != nil {
		fmt.Printf("failed creating file: %s", err)
		os.Exit(1)
	}
	csvwriter = csv.NewWriter(usergroups_csvfile)
	tmplen := 0
	csvwriter.Write(convert_to_slice_group_user(user_group_header))
	for _, zebedeegroup := range groupList.Teams {
		tmplen = tmplen + len(zebedeegroup.Members)
		for _, member := range zebedeegroup.Members {
			tmp := user_group{
				UserPoolId: "",
				Username:   member,
				GroupName:  zebedeegroup.Name,
			}
			csvwriter.Write(convert_to_slice_group_user(tmp))
		}

	}
	csvwriter.Flush()

	fmt.Println("There are ", tmplen, "records extracted to file", conf.groupusers_filename, "csv Errors ", csvwriter.Error())
	usergroups_csvfile.Close()
}
