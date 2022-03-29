update-topic-ids
==================

This utility updates `topics` documents to use a new nano id of size 4 using an alphabet of `123456789`.

See [decision record](https://github.com/ONSdigital/dp-decision-records/blob/feature/topics/TopicsService/0001-topic-and-subtopic-definition-and-assignment.md#root-and-sub-topic-id)

### How to run the utility

Run
```
mongo <mongo_url> <options> update-topic-ids.js
```

The `<mongo_url>` part should look like:
- `<host>:<port>/topics` for example: `localhost:27017/topics`
- If authentication is needed, use the format `mongodb://<username>:<password>@<host>:<port>/topics`
- in the above, `/topics` indicates the database to be modified

Example of the (optional) `<options>` part:

- `--eval 'cfg={verbose:true}'` (e.g. use for debugging)
  - `cfg` defaults to: `{verbose:false, update: true}`
  - if you specify `cfg`, all missing options default to `false`

Before updating a live database, it is recommended to perform a dry run and check the result looks as expected:

```
mongo localhost:27017/topics --eval 'cfg={verbose:true, update:false}' update-topic-ids.js
```

Note: when connecting to a TLS-enabled DocumentDB cluster (sandbox or prod), you'll need to add the following options:
- `--tls`
- `--tlsCAFile=<pem>` where `<pem>` is the path to the Certificate Authority .pem file

For example:
```
mongo mongodb://$MONGO_USER:$MONGO_PASS@$MONGO_HOST/topics --tls --tlsCAFile=./cert.pem update-topic-ids.js
```
