package main

import (
	"encoding/csv"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ONSdigital/dp-zebedee-sdk-go/zebedee"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/google/uuid"
)

const (
	logFileName = "Userlog"
	dateLayout  = "2006-01-02_15_04_05"
)

type config struct {
	environment,
	awsProfile,
	validUsersFileName,
	invalidUsersFileName,
	host,
	pword,
	user string
	emailDomains       []string
	s3Bucket, s3Region string
}

type cognitoUser struct {
	username            string `csv:"cognito:username"`
	name                string `csv:"name"`
	givenName           string `csv:"given_name"`
	familyName          string `csv:"family_name"`
	middleName          string `csv:"middle_name"`
	nickname            string `csv:"nickname"`
	preferredUsername   string `csv:"preferred_username"`
	profile             string `csv:"profile"`
	picture             string `csv:"picture"`
	website             string `csv:"website"`
	email               string `csv:"email"`
	emailVerified       string `csv:"email_verified"`
	gender              string `csv:"gender"`
	birthdate           string `csv:"birthdate"`
	zoneInfo            string `csv:"zoneinfo"`
	locale              string `csv:"locale"`
	phoneNumber         string `csv:"phone_number"`
	phoneNumberVerified string `csv:"phone_number_verified"`
	address             string `csv:"address"`
	updatedAt           string `csv:"updated_at"`
	mfaEnabled          string `csv:"cognito:mfa_enabled"`
	enabled             string `csv:"enabled"`
}

var header = cognitoUser{
	username:            "cognito:username",
	name:                "name",
	givenName:           "given_name",
	familyName:          "family_name",
	middleName:          "middle_name",
	nickname:            "nickname",
	preferredUsername:   "preferred_username",
	profile:             "profile",
	picture:             "picture",
	website:             "website",
	email:               "email",
	emailVerified:       "email_verified",
	gender:              "gender",
	birthdate:           "birthdate",
	zoneInfo:            "zoneinfo",
	locale:              "locale",
	phoneNumber:         "phone_number",
	phoneNumberVerified: "phone_number_verified",
	address:             "address",
	updatedAt:           "updated_at",
	mfaEnabled:          "cognito:mfa_enabled",
	enabled:             "enabled",
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
		case "invalidusers_filename":
			missingVariables("invalidusers_filename", pair[1])
			conf.invalidUsersFileName = pair[1]
		case "zebedee_user":
			missingVariables("zebedee_user", pair[1])
			conf.user = pair[1]
		case "zebedee_pword":
			missingVariables("zebedee_pword", pair[1])
			conf.pword = pair[1]
		case "zebedee_host":
			missingVariables("zebedee_host", pair[1])
			conf.host = pair[1]
		case "email_domains":
			missingVariables("email_domains", pair[1])
			conf.emailDomains = strings.Split(pair[1], ",")
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
	if len(value) == 0 {
		log.Fatalf("Please set Environment Variable: %s", envValue)
	}
	return true
}

func convertToSlice(input cognitoUser) []string {
	return []string{
		input.username,
		input.name,
		input.givenName,
		input.familyName,
		input.middleName,
		input.nickname,
		input.preferredUsername,
		input.profile,
		input.picture,
		input.website,
		input.email,
		input.emailVerified,
		input.gender,
		input.birthdate,
		input.zoneInfo,
		input.locale,
		input.phoneNumber,
		input.phoneNumberVerified,
		input.address,
		input.updatedAt,
		input.mfaEnabled,
		input.enabled,
	}
}

func processZebedeeUsers(validUsersWriter *csv.Writer, invalidUsersWriter *csv.Writer, userList []zebedee.User, validEmailDomains []string) (int, int) {
	var validUsersCount, invalidUsersCount int

	for _, user := range userList {
		var csvLine cognitoUser
		csvLine.username = uuid.NewString()
		csvLine.email = user.Email

		domain := strings.Split(user.Email, "@")
		names := strings.Split(domain[0], ".")

		if len(names) == 2 {
			csvLine.givenName = names[0]
			csvLine.familyName = names[1]
		} else if len(names) > 2 {
			csvLine.givenName = names[0]
			csvLine.familyName = names[2]
		} else {
			csvLine.givenName = ""
			csvLine.familyName = names[0]
		}

		csvLine.mfaEnabled = "false"
		csvLine.enabled = "true"
		csvLine.phoneNumberVerified = "false"
		csvLine.emailVerified = "true"

		userDetails := convertToSlice(csvLine)
		if validateEmailId(validEmailDomains, user.Email) {
			if err := validUsersWriter.Write(userDetails); err != nil {
				log.Printf("error writing record to csv: %v\n", err)
			} else {
				validUsersCount += 1
			}
		} else {
			if err := invalidUsersWriter.Write(userDetails); err != nil {
				log.Printf("error writing record to csv: %v\n", err)
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
		log.Fatalf("failed to open file %s, error is: %v", fileName, err)
		return err
	}

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s3Bucket),
		Key:    aws.String(s3FilePath),
		Body:   f,
	})
	if err != nil {
		log.Fatalf("failed to upload file %s, error is: %v", fileName, err)
		return err

	}
	log.Printf("file uploaded to %s\n", aws.StringValue(&result.Location))
	return nil
}

func ExtractUserData(conf *config) {
	httpCli := zebedee.NewHttpClient(time.Second * 5)
	zebCli := zebedee.NewClient(conf.host, httpCli)

	c := zebedee.Credentials{
		Email:    conf.user,
		Password: conf.pword,
	}

	sess, err := zebCli.OpenSession(c)
	if err != nil {
		log.Fatalf("zebedee open sessions, error is: %v", err)
	}

	userList, err := zebCli.GetUsers(sess)
	if err != nil {
		log.Fatalf("zebedee get users, error is: %v", err)

	}
	validUsersFile := createFile(conf.validUsersFileName)
	validUsersWriter := csv.NewWriter(validUsersFile)

	invalidUsersFile := createFile(conf.invalidUsersFileName)
	invalidUsersWriter := csv.NewWriter(invalidUsersFile)

	csvHeader := convertToSlice(header)
	err = validUsersWriter.Write(csvHeader)
	if err != nil {
		log.Fatalf("Theres been an issue in writing header to file %s, error is: %v", conf.validUsersFileName, err)
	}
	err = invalidUsersWriter.Write(csvHeader)
	if err != nil {
		log.Fatalf("Theres been an issue in writing header to file %s, error is: %v", conf.invalidUsersFileName, err)
	}

	validUsersCount, invalidUsersCount := processZebedeeUsers(validUsersWriter, invalidUsersWriter, userList, conf.emailDomains)
	validUsersWriter.Flush()
	invalidUsersWriter.Flush()

	log.Println("========= file validation =============")
	if validUsersCount+invalidUsersCount != len(userList) || validUsersWriter.Error() != nil || invalidUsersWriter.Error() != nil {
		log.Printf("There has been an error... \n")
		log.Printf("valid users writer Errors: %v\n", validUsersWriter.Error())
		log.Printf("invalid users writer Errors: %v\n", validUsersWriter.Error())
	}

	log.Println("Expected row count: - ", len(userList))
	log.Println("Valid users row count: - ", validUsersCount)
	log.Println("Invalid users row count: - ", invalidUsersCount)
	log.Println("=========")

	err = validUsersFile.Close()
	if err != nil {
		log.Printf("Theres been an issue in closing file %s, error is: %v", conf.validUsersFileName, err)
	}
	err = invalidUsersFile.Close()
	if err != nil {
		log.Printf("Theres been an issue in closing file %s, error is: %v", conf.invalidUsersFileName, err)

	}
	log.Println("========= Uploading valid users file to S3 =============")
	s3err := uploadFile(conf.validUsersFileName, conf.s3Bucket, conf.validUsersFileName, conf.s3Region, conf.awsProfile)
	if s3err != nil {
		log.Fatalf("Theres been an issue in uploading to s3 %v", err)
	}

	s3err = uploadFile(conf.invalidUsersFileName, conf.s3Bucket, conf.invalidUsersFileName, conf.s3Region, conf.awsProfile)
	if s3err != nil {
		log.Fatalf("Theres been an issue in uploading to s3 %v", err)
	}

	log.Println("========= Uploaded files to S3 =============")
	deleteFile(conf.invalidUsersFileName)

}

func createFile(fileName string) *os.File {
	csvFile, err := os.Create(fileName)
	if err != nil {
		log.Fatalf("failed creating file %s, error is: %v", fileName, err)
	}
	return csvFile
}

func deleteFile(fileName string) {
	err := os.Remove(fileName)
	if err != nil {
		log.Printf("failed deleting file %s, error is: %v", fileName, err)
	}
}
func main() {
	start := time.Now()
	conf := readConfig()
	now := time.Now().Format(dateLayout)
	logFileName := logFileName + "_" + now + ".log"
	logFileHandler, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("error opening file: %s, error is: %v", logFileName, err)
	}
	log.SetOutput(logFileHandler)
	log.SetFlags(log.LstdFlags | log.Lshortfile | log.Lmicroseconds)
	log.Printf("log file created, %s\n", logFileName)

	ExtractUserData(conf)
	elapsed := time.Since(start)
	log.Printf("Elapse time %s\n", elapsed)
	uploadFile(logFileName, conf.s3Bucket, logFileName, conf.s3Region, conf.awsProfile)
	deleteFile(logFileName)
}
