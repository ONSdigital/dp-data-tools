// delete a specified collection

db = db.getSiblingDB(DB_NAME)

const result = db[COLLECTION_NAME].drop()

if (result) {
    print(`collection '${COLLECTION_NAME}' in database '${DB_NAME}' deleted successfully`)
} else {
    print(`failed to delete collection '${COLLECTION_NAME}' in database '${DB_NAME}'`)
    quit(1);
}