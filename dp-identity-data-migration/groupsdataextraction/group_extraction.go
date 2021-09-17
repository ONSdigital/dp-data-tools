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
	groupsFilename,
	groupUsersFilename,
	host,
	pword,
	user string
}

func readConfig() (conf config) {
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)

		if pair[0] == "groups_filename" {
			conf.groupsFilename = pair[1]
		}

		if pair[0] == "groupusers_filename" {
			conf.groupUsersFilename = pair[1]
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

	if conf.host == "" || conf.pword == "" || conf.user == "" || conf.groupsFilename == "" || conf.groupUsersFilename == "" {
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

type userGroupCSV struct {
	Username string
	Groups   string
}

var headerUserGroup = userGroupCSV{
	Username: "user_name",
	Groups:   "groups",
}

func convertToSlice_Group(input group) []string {
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

func convertToSlice_UserGroup(input userGroupCSV) []string {
	return []string{
		input.Username,
		input.Groups,
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

var emptylist []string

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

	groupList, err := zebCli.ListTeams(sess)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	groupsCSVFile, err := os.Create(conf.groupsFilename)
	if err != nil {
		fmt.Printf("failed creating file: %s", err)
		os.Exit(1)
	}
	csvwriter := csv.NewWriter(groupsCSVFile)
	csvwriter.Write(convertToSlice_Group(group_header))

	for _, zebedeegroup := range groupList.Teams {
		tmp := group{
			GroupName:        fmt.Sprintf("%v", zebedeegroup.ID),
			UserPoolId:       "",
			Description:      zebedeegroup.Name,
			RoleArn:          "",
			Precedence:       "10",
			LastModifiedDate: "",
			CreationDate:     "",
		}
		csvwriter.Write(convertToSlice_Group(tmp))

	}
	csvwriter.Flush()
	groupsCSVFile.Close()
	fmt.Println("========= ", conf.groupsFilename, "file validiation =============")
	actualRowCount := readCsvFile(conf.groupsFilename) - 1
	if actualRowCount != len(groupList.Teams) || csvwriter.Error() != nil {
		fmt.Println("There has been an error... ")
		fmt.Println("csv Errors ", csvwriter.Error())
	}

	fmt.Println("Expected row count: - ", len(groupList.Teams))
	fmt.Println("Actual row count: - ", actualRowCount)
	fmt.Println("=========")

	tmpUserGroups := make(map[string][]string)

	usergroupsCSVFile, err := os.Create(conf.groupUsersFilename)
	if err != nil {
		fmt.Printf("failed creating file: %s", err)
		os.Exit(1)
	}
	csvwriter = csv.NewWriter(usergroupsCSVFile)
	csvwriter.Write(convertToSlice_UserGroup(headerUserGroup))

	userList, err := zebCli.GetUsers(sess)

	if err != nil {
		fmt.Println("Theres been an issue", err)
		os.Exit(1)
	}

	for _, user := range userList {

		_, isKeyPresent := tmpUserGroups[user.Email]
		if !isKeyPresent {
			tmpUserGroups[user.Email] = emptylist
		}

		permissions, err := zebCli.GetPermissions(sess, user.Email)
		if err != nil {
			fmt.Println(err)
		}
		if permissions.Admin {
			tmpUserGroups[user.Email] = append(tmpUserGroups[user.Email], "role-admin")
		}

		if permissions.Editor {
			tmpUserGroups[user.Email] = append(tmpUserGroups[user.Email], "role-publisher")
		}
	}

	for _, zebedeegroup := range groupList.Teams {

		for _, member := range zebedeegroup.Members {
			_, isKeyPresent := tmpUserGroups[member]
			if !isKeyPresent {
				fmt.Println("---")
				fmt.Println(member, "is not a user???")
				fmt.Println("---")
			} else {
				tmpUserGroups[member] = append(tmpUserGroups[member], fmt.Sprintf("%v", zebedeegroup.ID))
			}
		}
	}

	for k, v := range tmpUserGroups {
		tmp := userGroupCSV{
			Username: k,
			Groups:   strings.Join(v, ", "),
		}
		csvwriter.Write(convertToSlice_UserGroup(tmp))
	}

	csvwriter.Flush()
	usergroupsCSVFile.Close()

	actualRowCount = readCsvFile(conf.groupUsersFilename) - 1
	fmt.Println("========= ", conf.groupUsersFilename, "file validiation =============")
	if actualRowCount != len(userList) || csvwriter.Error() != nil {
		fmt.Println("There has been an error... ")
		fmt.Println("csv Errors ", csvwriter.Error())
	}

	fmt.Println("Expected row count: - ", len(userList))
	fmt.Println("Actual row count: - ", actualRowCount)
	fmt.Println("=========")
}
