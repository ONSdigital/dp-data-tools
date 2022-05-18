package main

import (
	"encoding/csv"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"os"

	"time"

	"github.com/ONSdigital/dp-zebedee-sdk-go/zebedee"
	uuid "github.com/google/uuid"
)

type config struct {
	validUsersFileName, invalidUsersFileName, host, pword, user string
	emailDomains                                                []string
	s3Bucket, s3BaseDir, s3Region                               string
}

func (c config) getS3ValidUsersFilePath() string {
	return fmt.Sprintf("%s%s", c.s3BaseDir, c.validUsersFileName)
}

func (c config) getS3InValidUsersFilePath() string {
	return fmt.Sprintf("%s%s", c.s3BaseDir, c.invalidUsersFileName)
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
	fmt.Println(conf)
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		switch pair[0] {
		case "filename":
			missing_variables("filename", pair[1])
			conf.validUsersFileName = pair[1]
			conf.invalidUsersFileName = fmt.Sprintf("invalid_%s", pair[1])
		case "zebedee_user":
			missing_variables("zebedee_user", pair[1])
			conf.user = pair[1]
		case "zebedee_pword":
			missing_variables("zebedee_pword", pair[1])
			conf.pword = pair[1]
		case "zebedee_host":
			missing_variables("zebedee_host", pair[1])
			conf.host = pair[1]
		case "email_domains":
			missing_variables("zebedee_host", pair[1])
			conf.emailDomains = strings.Split(pair[1], ",")
		case "s3_bucket":
			missing_variables("s3_bucket", pair[1])
			conf.s3Bucket = pair[1]
		case "s3_base_dir":
			missing_variables("s3_bucket", pair[1])
			conf.s3BaseDir = pair[1]
		case "s3_region":
			missing_variables("s3_region", pair[1])
			conf.s3Region = pair[1]
		}
	}
	if conf.host == "" || conf.pword == "" || conf.user == "" || conf.validUsersFileName == "" {
		fmt.Println("Please set Environment Variables ")
		os.Exit(1)
	}

	return conf
}

func missing_variables(envValue string, value string) bool {
	if len(value) == 0 {
		fmt.Println("Please set Environment Variables ", envValue)
		os.Exit(1)
	}
	return true
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

func process_zebedee_users(validUsersWriter *csv.Writer, invalidUsersWriter *csv.Writer, userList []zebedee.User, validEmailDomains []string) (int, int) {
	var validUsersCount, invalidUsersCount int

	for _, user := range userList {
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

		userDetails := convert_to_slice(csvline)
		if validateEmailId(validEmailDomains, user.Email) {
			if err := validUsersWriter.Write(userDetails); err != nil {
				fmt.Println("error writing record to csv:", err)
			} else {
				validUsersCount += 1
			}
		} else {
			if err := invalidUsersWriter.Write(userDetails); err != nil {
				fmt.Println("error writing record to csv:", err)
			} else {
				invalidUsersCount += 1
			}
		}
	}
	return validUsersCount, invalidUsersCount
}

func validateEmailId(validEmailDomains []string, emailID string) bool {
	if strings.Contains(emailID, "@") {
		domainName := strings.Split(emailID, "@")[1]
		for _, domain := range validEmailDomains {
			if domain == domainName {
				return true
			}
		}
	}
	return false
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
		return fmt.Errorf("failed to upload file: %+v", err)
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

	userList, err := zebCli.GetUsers(sess)

	if err != nil {
		fmt.Println("Theres been an issue")
		fmt.Println(err)
		os.Exit(1)
	}

	validUsersFile := createFile(conf.validUsersFileName)
	validUsersWriter := csv.NewWriter(validUsersFile)

	invalidUsersFile := createFile(conf.invalidUsersFileName)
	invalidUsersWriter := csv.NewWriter(invalidUsersFile)

	csvheader := convert_to_slice(header)
	validUsersWriter.Write(csvheader)
	invalidUsersWriter.Write(csvheader)

	validUsersCount, invalidUsersCount := process_zebedee_users(validUsersWriter, invalidUsersWriter, userList, conf.emailDomains)
	validUsersWriter.Flush()
	invalidUsersWriter.Flush()

	fmt.Println("========= file validiation =============")
	if validUsersCount+invalidUsersCount != len(userList) || validUsersWriter.Error() != nil || invalidUsersWriter.Error() != nil {
		fmt.Println("There has been an error... ")
		fmt.Println("valid users writer Errors ", validUsersWriter.Error())
		fmt.Println("invalid users writer Errors ", validUsersWriter.Error())
	}

	fmt.Println("Expected row count: - ", len(userList))
	fmt.Println("Valid users row count: - ", validUsersCount)
	fmt.Println("Invalid users row count: - ", invalidUsersCount)
	fmt.Println("=========")

	validUsersFile.Close()
	invalidUsersFile.Close()

	fmt.Println("========= Uploading valid users file to S3 =============")
	uploadFile(conf.validUsersFileName, conf.s3Bucket, conf.getS3ValidUsersFilePath(), conf.s3Region)
	uploadFile(conf.invalidUsersFileName, conf.s3Bucket, conf.getS3InValidUsersFilePath(), conf.s3Region)
	fmt.Println("========= Uploaded fules to S3 =============")

}

func createFile(fileName string) *os.File {
	csvFile, err := os.Create(fileName)
	if err != nil {
		fmt.Printf("failed creating file: %s", err)
		os.Exit(1)
	}
	return csvFile
}
