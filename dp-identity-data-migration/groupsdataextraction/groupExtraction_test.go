package groupsdataextraction

import (
	"fmt"

	"os"
	"testing"
	"time"

	"github.com/ONSdigital/dp-zebedee-sdk-go/zebedee"
	. "github.com/smartystreets/goconvey/convey"
)

func TestZebedeeClient_ListTeams(t *testing.T) {
	Convey("test 200", t, func() {

		httpCli := zebedee.NewHttpClient(time.Second * 5)
		zebCli := zebedee.NewClient("http://localhost:8082", httpCli)

		c := zebedee.Credentials{
			Email:    "ann.witcher@ons.gov.uk",
			Password: "R0s3 R3d Sn0w Wh1t3",
		}

		sess, err := zebCli.OpenSession(c)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// groups process
		groupList, err := zebCli.ListTeams(sess)
		So(err, ShouldBeNil)
		So(groupList, ShouldNotBeNil)
	})

}
