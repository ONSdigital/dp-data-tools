package main

import (
	"encoding/csv"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ONSdigital/dp-zebedee-sdk-go/zebedee"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type config struct {
	environment,
	groupsFilename,
	groupUsersFilename,
	host,
	pword,
	user,
	s3Bucket,
	s3Region string
}

func ExtractGroupsData() {

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
		fmt.Println("Get data from zebedee", err)
		os.Exit(1)
	}

	userList, err := zebCli.GetUsers(sess)
	if err != nil {
		fmt.Println("Theres been an issue", err)
		os.Exit(1)
	}
	tmpUserGroups := make(map[string][]string)
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

	conf.processGroups(groupList)
	conf.processGroupsUsers(groupList, userList, tmpUserGroups)
}

func (c config) processGroups(groupList zebedee.TeamsList) {
	groupsCSVFile, err := os.Create(c.groupsFilename)
	if err != nil {
		fmt.Printf("failed creating file: %s", err)
		os.Exit(1)
	}

	csvwriter := csv.NewWriter(groupsCSVFile)
	if write_err := csvwriter.Write(convertToSlice_Group(group_header)); write_err != nil {
		fmt.Printf("failed writing file: %s", err)
		os.Exit(1)
	}

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
		if write_err := csvwriter.Write(convertToSlice_Group(tmp)); write_err != nil {
			fmt.Printf("failed writing file: %s", err)
			os.Exit(1)
		}
	}
	csvwriter.Flush()
	groupsCSVFile.Close()

	fmt.Println("========= ", c.groupsFilename, "file validiation =============")
	f, err := os.Open(c.groupsFilename)
	if err != nil {
		fmt.Printf("failed opening file: %s", err)
		os.Exit(1)
	}
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		fmt.Printf("failed reading file: %s", err)
		os.Exit(1)
	}

	if len(records)-1 != len(groupList.Teams) || csvwriter.Error() != nil {
		fmt.Println("There has been an error... ")
		fmt.Println("csv Errors ", csvwriter.Error())
		fmt.Println(len(records)-1, len(records))
	}

	fmt.Println("Expected row count: - ", len(groupList.Teams))
	fmt.Println("Actual row count: - ", len(records)-1)
	fmt.Println("=========")

	fmt.Println("Uploading", c.environment+"/"+c.groupsFilename, "to s3")

	s3err := uploadFile(c.groupsFilename, c.s3Bucket, c.environment+"/"+c.groupsFilename, c.s3Region)
	if s3err != nil {
		fmt.Println("Theres been an issue in uploading to s3")
		fmt.Println(s3err)
		os.Exit(1)
	}
	fmt.Println("Uploaded", c.groupsFilename, "to s3")

	deleteFile(c.groupsFilename)
}

func (c config) processGroupsUsers(groupList zebedee.TeamsList, userList []zebedee.User, userRoles map[string][]string) {

	usergroupsCSVFile, err := os.Create(c.groupUsersFilename)
	if err != nil {
		fmt.Printf("failed creating file: %s", err)
		os.Exit(1)
	}
	csvwriter := csv.NewWriter(usergroupsCSVFile)
	if write_err := csvwriter.Write(convertToSlice_UserGroup(headerUserGroup)); write_err != nil {
		fmt.Printf("failed writing file: %s", err)
		os.Exit(1)
	}
	for _, zebedeegroup := range groupList.Teams {
		for _, member := range zebedeegroup.Members {
			_, isKeyPresent := userRoles[member]
			if !isKeyPresent {
				fmt.Println("---")
				fmt.Println(member, "is not a user???")
				fmt.Println("---")
			} else {
				userRoles[member] = append(userRoles[member], fmt.Sprintf("%v", zebedeegroup.ID))
			}
		}
	}

	for k, v := range userRoles {
		tmp := userGroupCSV{
			Username: k,
			Groups:   strings.Join(v, ", "),
		}
		csvwriter.Write(convertToSlice_UserGroup(tmp))
	}
	csvwriter.Flush()
	usergroupsCSVFile.Close()

	f, err := os.Open(c.groupUsersFilename)
	if err != nil {
		fmt.Printf("failed opening file: %s", err)
		os.Exit(1)
	}
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		fmt.Printf("failed reading file: %s", err)
		os.Exit(1)
	}

	fmt.Println("========= ", c.groupUsersFilename, "file validation =============")
	if len(records)-1 != len(userList) || csvwriter.Error() != nil {
		fmt.Println("There has been an error... ")
		fmt.Println("csv Errors ", csvwriter.Error())
	}

	fmt.Println("Expected row count: - ", len(userList))
	fmt.Println("Actual row count: - ", len(records)-1)
	fmt.Println("=========")

	fmt.Println("Uploading", c.environment+"/"+c.groupUsersFilename, "to s3")

	s3err := uploadFile(c.groupUsersFilename, c.s3Bucket, c.environment+"/"+c.groupUsersFilename, c.s3Region)
	if s3err != nil {
		fmt.Println("Theres been an issue in uploading to s3")
		fmt.Println(s3err)
		os.Exit(1)
	}
	fmt.Println("Uploaded", c.groupUsersFilename, "to s3")

	deleteFile(c.groupUsersFilename)
}

func readConfig() *config {
	conf := &config{}
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		switch pair[0] {
		case "environment":
			conf.environment = pair[1]
		case "groups_filename":
			conf.groupsFilename = pair[1]
		case "groupusers_filename":
			conf.groupUsersFilename = pair[1]
		case "zebedee_user":
			conf.user = pair[1]
		case "zebedee_pword":
			conf.pword = pair[1]
		case "zebedee_host":
			conf.host = pair[1]
		case "s3_bucket":
			conf.s3Bucket = pair[1]
		case "s3_region":
			conf.s3Region = pair[1]
		}
	}

	missing_variables("environment", conf.environment)
	missing_variables("groups_filename", conf.groupsFilename)
	missing_variables("groupusers_filename", conf.groupUsersFilename)
	missing_variables("zebedee_user", conf.user)
	missing_variables("zebedee_pword", conf.pword)
	missing_variables("zebedee_host", conf.host)
	missing_variables("s3_bucket", conf.s3Bucket)
	missing_variables("s3_region", conf.s3Region)

	return conf
}

func missing_variables(envValue string, value string) bool {
	if len(value) == 0 {
		fmt.Println("Please set Environment Variables ", envValue)
		os.Exit(3)
	}
	return true
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

func uploadFile(fileName, s3Bucket, s3FilePath, region string) error {
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

func deleteFile(fileName string) {
	err := os.Remove(fileName)
	if err != nil {
		fmt.Printf("failed deleting file: %s", err)
		os.Exit(1)
	}
}

func main() {
	start := time.Now()
	ExtractGroupsData()
	elapsed := time.Since(start)
	fmt.Printf("Elapse time %s\n", elapsed)
}
