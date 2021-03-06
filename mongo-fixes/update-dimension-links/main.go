package main

import (
	"context"
	"errors"
	"flag"
	"strings"
	"time"

	"github.com/ONSdigital/dp-dataset-api/models"
	"github.com/ONSdigital/log.go/log"
	"github.com/globalsign/mgo"
	"github.com/globalsign/mgo/bson"
	"github.com/schollz/progressbar/v3"
)

var (
	mongoURL string
)

// MongoID represents instance id
type MongoID struct {
	ID bson.ObjectId `bson:"_id"`
}

func main() {
	flag.StringVar(&mongoURL, "mongo-url", mongoURL, "mongoDB URL")
	flag.Parse()

	ctx := context.Background()

	if mongoURL == "" {
		log.Event(ctx, "missing mongo-url flag", log.ERROR)
		return
	}

	session, err := mgo.Dial(mongoURL)
	if err != nil {
		log.Event(ctx, "unable to create mongo session", log.Error(err), log.ERROR)
		return
	}
	defer session.Close()

	session.SetBatch(10000)
	session.SetPrefetch(0.25)

	log.Event(ctx, "successfully connected to mongo", log.INFO)

	// Get all instances IDs
	instanceIDs, err := getInstanceIDs(ctx, session)
	if err != nil {
		log.Event(ctx, "failed to get all instances", log.Error(err), log.ERROR)
		return
	}
	log.Event(ctx, "successfully retrieved all instance ids", log.INFO)

	// Create a backup collection
	// dateTime formatted in YYYYMMDD_HHMMSS
	dateTime := time.Now().Format("20060102_150405")
	backupProgressBar := progressbar.Default(int64(len(instanceIDs)), "backup instance")
	for _, id := range instanceIDs {
		if err = addInstanceToBackup(ctx, session, id.ID, dateTime); err != nil {
			log.Event(ctx, "failed to backup instances", log.Error(err), log.ERROR)
			return
		}
		backupProgressBar.Add(1)
		time.Sleep(40 * time.Millisecond)
	}

	log.Event(ctx, "successfully backed up all instance documents", log.INFO)

	errorCount := 0
	updateProgressBar := progressbar.Default(int64(len(instanceIDs)), "updating instance")

	// loop over instances
	for _, id := range instanceIDs {

		// update the whole instance document
		if err := updateInstance(ctx, session, id.ID); err != nil {
			log.Event(ctx, "failed to update dimension of instance", log.Error(err), log.Data{"current_instance_id": id}, log.ERROR)
			errorCount++
			if errorCount > 10 {
				log.Event(ctx, "too many errors updating instances", log.Error(err), log.ERROR)
				return
			}
		}

		updateProgressBar.Add(1)
		time.Sleep(40 * time.Millisecond)
	}

	if errorCount > 0 {
		log.Event(ctx, "failed to update dimension of all instances", log.Data{"unsuccessful_update_count": errorCount}, log.WARN)
	} else {
		log.Event(ctx, "successfully updated all instance documents", log.INFO)
	}

}

func getInstanceIDs(ctx context.Context, session *mgo.Session) (results []MongoID, err error) {
	s := session.Copy()
	defer s.Close()

	err = s.DB("datasets").C("instances").Find(bson.M{"dimensions": bson.M{"$elemMatch": bson.M{"href": bson.M{"$regex": "/v1/"}}}}).Select(bson.M{"_id": 1}).All(&results)
	if err != nil {
		log.Event(ctx, "failed to get instance ids", log.Error(err), log.ERROR)
		return nil, err
	}

	if len(results) < 1 {
		return nil, errors.New("no instance documents found")
	}

	return results, nil
}

// addInstanceToBackup backs-up an instance document
func addInstanceToBackup(ctx context.Context, session *mgo.Session, id bson.ObjectId, dateTime string) error {

	s := session.Copy()
	defer s.Close()

	var instance models.Instance
	err := s.DB("datasets").C("instances").Find(bson.M{"_id": id}).One(&instance)
	if err != nil {
		log.Event(ctx, "failed to get instance from id", log.Error(err), log.ERROR)
		return err
	}

	_, err = s.DB("datasets").C("instances_backup_"+dateTime).UpsertId(id, instance)
	if err != nil {
		log.Event(ctx, "failed to add instance to backup", log.Error(err), log.ERROR)
		return err
	}

	return nil
}

// updateInstance updates an instance document
func updateInstance(ctx context.Context, session *mgo.Session, id bson.ObjectId) (err error) {
	s := session.Copy()
	defer s.Close()

	var instance models.Instance
	err = s.DB("datasets").C("instances").Find(bson.M{"_id": id}).One(&instance)
	if err != nil {
		log.Event(ctx, "failed to get instance from id for updating", log.Error(err), log.ERROR)
		return err
	}

	// loop over dimensions
	for i, dimension := range instance.Dimensions {
		for strings.Contains(dimension.HRef, "/v1/") {
			dimension.HRef = strings.Replace(dimension.HRef, "/v1/", "/", 1)
		}
		instance.Dimensions[i].HRef = dimension.HRef
	}

	// prepares updated_instance in bson.M and then updates existing instance document
	updateInstance := bson.M{"$set": instance}

	err = s.DB("datasets").C("instances").Update(bson.M{"_id": id}, updateInstance)
	if err != nil {
		if err == mgo.ErrNotFound {
			return errors.New("instance not found")
		}
	}
	return err
}
