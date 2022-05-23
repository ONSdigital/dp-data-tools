# dp-data-tools

 See below for various tools to update data.

## Prerequisites

Some of these tools require [dp-cli](https://github.com/ONSdigital/dp-cli).

## Current tools/scripts

### mongodb related

* [Edition document restructure](./mongo-fixes/edition-doc-structure)
* [Filter blueprint and output documents include new dataset object](./mongo-fixes/filter-doc-version-identifier)
* [Instance/version documents include new downloads structure](./mongo-fixes/download-structure/dataset)
* [Filter output documents include new downloads structure](./mongo-fixes/download-structure/filter)
* [Remove collection_id from published datasets](./mongo-fixes/delete-published-collection-id)
* [Neptune migration - clear all collections and import updated recipes](./mongo-fixes/neptune-migration)
* [Copy datasets from mongodb on develop to local mongodb](./mongo-tools/copy-datasets)
* [Update topics IDs](./mongo-fixes/update-topic-ids)

### kafka related

* [Drain a kafka topic of messages](./kafka-tools/drain-topic)
* [Check audit messages have been added to kafka](./kafka-tools/check-audit)
* [Queue a kafka message to rebuild full downloads for a dataset](./kafka-tools/generate-downloads)

### dp-topics-api related

* [Generate Topics database](./topics-tools/gen-topics-database)
* [Update topics IDs](./mongo-fixes/update-topic-ids)