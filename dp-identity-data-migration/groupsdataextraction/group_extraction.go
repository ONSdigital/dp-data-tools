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
		fmt.Println(conf.host)

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

func readCsvFile(filePath string) int {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Println("Unable to read input file "+filePath, err)
	}
	defer f.Close()

	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		fmt.Println("Unable to parse file as CSV for "+filePath, err)
	}

	return len(records)
}

func getClient(conf config) zebedee.Client {
	httpCli := zebedee.NewHttpClient(time.Second * 5)
	return zebedee.NewClient(conf.host, httpCli)
}

func main() {

	conf := readConfig()
	zebCli := getClient(conf)

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
	groups_csvfile.Close()

	actualrowcount := readCsvFile(conf.groups_filename) - 1
	if actualrowcount == len(groupList.Teams) || csvwriter.Error() == nil {
		fmt.Println(conf.groups_filename)
		fmt.Println("Expected row count: - ", len(groupList.Teams))
		fmt.Println("Actual row count: - ", actualrowcount)
		fmt.Println("csv Errors ", csvwriter.Error())
	} else {
		fmt.Println(conf.groups_filename)
		fmt.Println("There has been an error... ")
		fmt.Println("Expected row count: - ", len(groupList.Teams))
		fmt.Println("Actual row count: - ", actualrowcount)
		fmt.Println("csv Errors ", csvwriter.Error())
	}

	tmplen := 0
	usergroups_csvfile, err := os.Create(conf.groupusers_filename)
	if err != nil {
		fmt.Printf("failed creating file: %s", err)
		os.Exit(1)
	}
	csvwriter = csv.NewWriter(usergroups_csvfile)
	csvwriter.Write(convert_to_slice_group_user(user_group_header))

	userList, err := getUsers(zebCli, sess)
	if err != nil {
		fmt.Println("Theres been an issue", err)
		os.Exit(1)
	}

	for _, user := range userList {
		permissions, err := zebCli.GetPermissions(sess, user.Email)
		if err != nil {
			fmt.Println(err)
		}
		if permissions.Admin {
			tmp := user_group{
				UserPoolId: "",
				Username:   user.Email,
				GroupName:  "role-admin",
			}
			csvwriter.Write(convert_to_slice_group_user(tmp))
			tmplen += 1
		}

		if permissions.Editor {
			tmp := user_group{
				UserPoolId: "",
				Username:   user.Email,
				GroupName:  "role-publisher",
			}
			csvwriter.Write(convert_to_slice_group_user(tmp))
			tmplen += 1
		}

	}

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
	usergroups_csvfile.Close()

	fmt.Println("There are expected rows ", tmplen, ", actual rows", readCsvFile(conf.groupusers_filename), "records extracted to file", conf.groupusers_filename, "csv Errors ", csvwriter.Error())
	usergroups_csvfile.Close()

	actualrowcount = readCsvFile(conf.groupusers_filename) - 1
	if actualrowcount == tmplen || csvwriter.Error() == nil {
		fmt.Println(conf.groupusers_filename)
		fmt.Println("Expected row count: - ", tmplen)
		fmt.Println("Actual row count: - ", actualrowcount)
		fmt.Println("csv Errors ", csvwriter.Error())
	} else {
		fmt.Println(conf.groupusers_filename)
		fmt.Println("There has been an error... ")
		fmt.Println("Expected row count: - ", tmplen)
		fmt.Println("Actual row count: - ", actualrowcount)
		fmt.Println("csv Errors ", csvwriter.Error())
	}

}