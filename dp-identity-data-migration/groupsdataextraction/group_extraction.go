package main

import (
	"encoding/csv"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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
	user,
	s3Bucket,
	s3BaseDir,
	s3Region string
}

func (c config) getS3GroupsFilePath() string {
	return fmt.Sprintf("%s%s", c.s3BaseDir, c.groupsFilename)
}

func (c config) getS3GroupUsersFilePath() string {
	return fmt.Sprintf("%s%s", c.s3BaseDir, c.groupUsersFilename)
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
		if pair[0] == "s3_bucket" {
			conf.s3Bucket = pair[1]
		}
		if pair[0] == "s3_base_dir" {
			conf.s3BaseDir = pair[1]
		}
		if pair[0] == "s3_region" {
			conf.s3Region = pair[1]
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

func uploadFile(fileName , s3Bucket, s3FilePath, region string) error {
	sess := session.Must(session.NewSession(&aws.Config{Region: aws.String(region)}))

	uploader := s3manager.NewUploader(sess)

	f, err := os.Open(fileName)
	if err != nil {
		return fmt.Errorf("failed to open file %q, %+v", fileName, err)
	}

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3FilePath),
		Body:   f,
	})
	if err != nil {
		return fmt.Errorf("failed to upload file, %+v", err)
	}
	fmt.Printf("file uploaded to, %s\n", aws.StringValue(&result.Location))
	return nil
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

	// groups process
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

	fmt.Println("========= ", conf.groupsFilename, "file validiation =============")
	records, _ := csv.NewReader(groupsCSVFile).ReadAll()
	actualRowCount := len(records) - 1
	if actualRowCount != len(groupList.Teams) || csvwriter.Error() != nil {
		fmt.Println("There has been an error... ")
		fmt.Println("csv Errors ", csvwriter.Error())
	}

	fmt.Println("Expected row count: - ", len(groupList.Teams))
	fmt.Println("Actual row count: - ", actualRowCount)
	fmt.Println("=========")

	csvwriter.Flush()
	groupsCSVFile.Close()

	fmt.Println("Uploading", conf.groupsFilename, "to s3")

	uploadFile(conf.groupsFilename, conf.s3Bucket, conf.getS3GroupsFilePath(), conf.s3Region)

	fmt.Println("Uploaded", conf.groupsFilename, "to s3")
	// UserGroups part...

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
			tmpUserGroups[user.Email] = make([]string, 0)
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

	records, _ = csv.NewReader(usergroupsCSVFile).ReadAll()
	actualRowCount = len(records)
	fmt.Println("========= ", conf.groupUsersFilename, "file validation =============")
	if actualRowCount != len(userList) || csvwriter.Error() != nil {
		fmt.Println("There has been an error... ")
		fmt.Println("csv Errors ", csvwriter.Error())
	}

	fmt.Println("Expected row count: - ", len(userList))
	fmt.Println("Actual row count: - ", actualRowCount)
	fmt.Println("=========")

	csvwriter.Flush()
	usergroupsCSVFile.Close()

	fmt.Println("Uploading", conf.groupUsersFilename, "to s3")

	uploadFile(conf.groupUsersFilename, conf.s3Bucket, conf.getS3GroupUsersFilePath(), conf.s3Region)

	fmt.Println("Uploaded", conf.groupUsersFilename, "to s3")

}
