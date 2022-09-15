package main

import (
	"encoding/csv"
	"fmt"
	"github.com/google/uuid"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ONSdigital/dp-zebedee-sdk-go/zebedee"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const (
	logFileName = "groupslog"
	dateLayout  = "2006-01-02_15_04_05"
)

type config struct {
	environment,
	awsProfile,
	groupsFilename,
	groupUsersFilename,
	validUsersFileName,
	host,
	pword,
	user,
	s3Bucket,
	s3Region string
}

type amendedGroupList struct {
	ID               string
	cognitoGroupName string
	Name             string
	Members          []string
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
		log.Fatal("zebedee open sessions", err)
	}

	// get teams from zebedee
	groupList, err := zebCli.ListTeams(sess)
	if err != nil {
		log.Fatal("Get data from zebedee", err)
	}

	// get valid users from user_extract
	usersCSV, err := os.Open(conf.validUsersFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer usersCSV.Close()
	userReader := csv.NewReader(usersCSV)
	userReader.Read()
	rows, err := userReader.ReadAll()
	if err != nil {
		log.Fatal("Cannot read CSV file:", err)
	}
	users := map[string]string{}
	for _, row := range rows {
		users[row[10]] = row[0]
	}

	tmpUserGroups := make(map[string][]string)
	for k := range users {
		_, isKeyPresent := tmpUserGroups[k]
		if !isKeyPresent {
			tmpUserGroups[k] = make([]string, 0)
		}
		permissions, err := zebCli.GetPermissions(sess, k)
		if err != nil {
			log.Println(err)
		}
		if permissions.Admin {
			tmpUserGroups[k] = append(tmpUserGroups[k], "role-admin")
		}

		if permissions.Editor {
			tmpUserGroups[k] = append(tmpUserGroups[k], "role-publisher")
		}
	}

	amendedGroupList := conf.processGroups(groupList)
	conf.processGroupsUsers(amendedGroupList, users, tmpUserGroups)
}

func (c config) processGroups(groupList zebedee.TeamsList) []amendedGroupList {
	var returnList []amendedGroupList
	groupsCSVFile, err := os.Create(c.groupsFilename)
	if err != nil {
		log.Fatal("failed creating file: %s", err)
	}

	csvwriter := csv.NewWriter(groupsCSVFile)
	if write_err := csvwriter.Write(convertToSlice_Group(group_header)); write_err != nil {
		log.Fatal("failed writing file: %s", err)
	}
	for _, zebedeegroup := range groupList.Teams {
		var tmp = group{
			GroupName:        uuid.NewString(),
			UserPoolId:       "",
			Description:      zebedeegroup.Name,
			RoleArn:          "",
			Precedence:       "10",
			LastModifiedDate: "",
			CreationDate:     "",
		}
		var tmpReturn = amendedGroupList{
			ID:               zebedeegroup.ID,
			cognitoGroupName: tmp.GroupName,
			Name:             zebedeegroup.Name,
			Members:          zebedeegroup.Members,
		}
		returnList = append(returnList, tmpReturn)
		if write_err := csvwriter.Write(convertToSlice_Group(tmp)); write_err != nil {
			log.Fatal("failed writing file: %s", err)
		}

	}
	csvwriter.Flush()
	err = groupsCSVFile.Close()
	if err != nil {
		log.Fatal("failed closing file: %s", err)
	}

	log.Println("========= ", c.groupsFilename, "file validiation =============")
	f, err := os.Open(c.groupsFilename)
	if err != nil {
		log.Fatal("failed opening file: %s", err)
	}
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		log.Fatal("failed reading file: %s", err)
	}

	if len(records)-1 != len(groupList.Teams) || csvwriter.Error() != nil {
		log.Println("There has been an error... ")
		log.Println("csv Errors ", csvwriter.Error())
		log.Println(len(records)-1, len(records))
	}

	log.Println("Expected row count: - ", len(groupList.Teams))
	log.Println("Actual row count: - ", len(records)-1)
	log.Println("=========")

	log.Println("Uploading", c.groupsFilename, "to s3")

	s3err := uploadFile(c.groupsFilename, c.s3Bucket, c.groupsFilename, c.s3Region, c.awsProfile)
	if s3err != nil {
		log.Fatal("Theres been an issue in uploading to s3 %v", s3err)
	} else {
		log.Println("Uploaded", c.groupsFilename, "to s3")
	}
	return returnList
}

func (c config) processGroupsUsers(groupList []amendedGroupList, userList map[string]string, userRoles map[string][]string) {
	usergroupsCSVFile, err := os.Create(c.groupUsersFilename)
	if err != nil {
		log.Fatal("failed creating file: %s", err)
	}
	csvwriter := csv.NewWriter(usergroupsCSVFile)
	if write_err := csvwriter.Write(convertToSlice_UserGroup(headerUserGroup)); write_err != nil {
		log.Fatal("failed writing file: %s", err)
	}
	for _, zebedeegroup := range groupList {
		for _, member := range zebedeegroup.Members {
			_, isKeyPresent := userRoles[member]
			if isKeyPresent {
				userRoles[member] = append(userRoles[member], fmt.Sprintf("%v", zebedeegroup.cognitoGroupName))
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
		log.Fatal("failed opening file: %s", err)
	}
	records, err := csv.NewReader(f).ReadAll()
	if err != nil {
		log.Fatal("failed reading file: %s", err)
	}

	log.Println("========= ", c.groupUsersFilename, "file validation =============")
	if len(records)-1 != len(userList) || csvwriter.Error() != nil {
		log.Fatal("csv Errors ", csvwriter.Error())
	}

	log.Println("Expected row count: - ", len(userList))
	log.Println("Actual row count: - ", len(records)-1)
	log.Println("=========")

	log.Println("Uploading", c.groupUsersFilename, "to s3")

	s3err := uploadFile(c.groupUsersFilename, c.s3Bucket, c.groupUsersFilename, c.s3Region, c.awsProfile)
	if s3err != nil {
		log.Fatal("Theres been an issue in uploading to s3", s3err)
	} else {
		log.Println("Uploaded", c.groupUsersFilename, "to s3")

		deleteFile(c.groupsFilename)
		deleteFile(c.groupUsersFilename)
		deleteFile(c.validUsersFileName)

	}
}

func readConfig() *config {
	conf := &config{}
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		switch pair[0] {
		case "environment":
			missingVariables("environment", pair[1])
			conf.environment = pair[1]
		case "aws_profile":
			missingVariables("aws_profile", pair[1])
			conf.awsProfile = pair[1]
		case "validusers_filename":
			missingVariables("validusers_filename", pair[1])
			conf.validUsersFileName = pair[1]
		case "groups_filename":
			missingVariables("groups_filename", pair[1])
			conf.groupsFilename = pair[1]
		case "groupusers_filename":
			missingVariables("groupusers_filename", pair[1])
			conf.groupUsersFilename = pair[1]
		case "zebedee_user":
			missingVariables("zebedee_user", pair[1])
			conf.user = pair[1]
		case "zebedee_pword":
			missingVariables("zebedee_pword", pair[1])
			conf.pword = pair[1]
		case "zebedee_host":
			missingVariables("zebedee_host", pair[1])
			conf.host = pair[1]
		case "s3_bucket":
			missingVariables("s3_bucket", pair[1])
			conf.s3Bucket = pair[1]
		case "s3_region":
			missingVariables("s3_region", pair[1])
			conf.s3Region = pair[1]
		}
	}

	return conf
}

func missingVariables(envValue string, value string) bool {
	if len(value) < 1 {
		log.Fatal("Please set Environment Variables ", envValue)
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

func uploadFile(fileName, s3Bucket, s3FilePath, region, awsProfile string) error {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		Profile: awsProfile,
		Config: aws.Config{
			Region: aws.String(region),
		},
		SharedConfigState: session.SharedConfigEnable,
	}))
	uploader := s3manager.NewUploader(sess)

	f, err := os.Open(fileName)
	if err != nil {
		log.Fatal("failed to open file %q, %+v", fileName, err)
		return err
	}

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3FilePath),
		Body:   f,
	})
	if err != nil {
		log.Fatal("failed to upload file, %+v", err)
		return err
	}
	log.Printf("file uploaded to, %s\n", aws.StringValue(&result.Location))
	return nil
}

func deleteFile(fileName string) {
	err := os.Remove(fileName)
	if err != nil {
		log.Fatal("failed deleting file: %s", err)
	}
}

func main() {
	start := time.Now()
	conf := readConfig()
	now := time.Now().Format(dateLayout)

	logFileName := logFileName + "_" + now + ".log"
	logFileHandler, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal("error opening file: %v", err)
	}
	log.SetOutput(logFileHandler)
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	log.Println("log file created")

	ExtractGroupsData()

	elapsed := time.Since(start)
	log.Printf("Elapse time %s\n", elapsed)
	uploadFile(logFileName, conf.s3Bucket, logFileName, conf.s3Region, conf.awsProfile)
	deleteFile(logFileName)
}
