// update-topic-id.js
//
// Update all topic ids to use a new nano id

const collection = 'topics'
const idSize = 4
const idAlphabet = '123456789'

if (typeof(cfg) == "undefined") {
    // default, but can be changed on command-line, see README
    cfg = {
        verbose:  false,    // do a find first, show what would be changed
        update:   true      // set to false to avoid updates
    }
}

//////////////////////////

var idMap = new Object() // Store oldId => newId
var allNewIds = []  // Store all new ids to check for clashes
function getNewId(oldId) {
    if (idMap[oldId] == null) {
        do {
            newId = makeId()
        } while (isUsedId(newId))
        allNewIds.push(newId)
        idMap[oldId] = newId
        print(oldId + " becomes " + newId)
    }
    return idMap[oldId]
}

function isUsedId(id) {
    for (var i in allNewIds) {
        if (i == id) {
            return true
        }
    }
    return false
}

function makeId() {
    var result = ''
    for (var i = 0; i < idSize; i++) {
      result += idAlphabet.charAt(Math.floor(Math.random() * idAlphabet.length))
    }
    return result
}

function updateTopicDocument(topic) {
    var oldId = topic.id
    if (oldId != 'topic_root') {
        var newId = getNewId(topic.id)

        topic.id = newId
        topic.next.id = newId
        topic.current.id = newId
    
        updateLinks(topic.next.links, oldId, newId)
        updateLinks(topic.current.links, oldId, newId)
    }

    updateSubtopics(topic.next)
    updateSubtopics(topic.current)
}

function updateLinks(links, oldId, newId) {
    if (links) {
        var regex = new RegExp("/"+oldId)
        if (links.self) {
            links.self.id = newId
            links.self.href = links.self.href.replace(regex, "/"+newId)
        }
        if (links.content) {
            links.content.href = links.content.href.replace(regex, "/"+newId)
        }
        if (links.subtopics) {
            links.subtopics.href = links.subtopics.href.replace(regex, "/"+newId)
        }
    }
}

function updateSubtopics(element) {
    if (element.subtopics_ids) {
        for (var i = 0; i < element.subtopics_ids.length; i++) {
            element.subtopics_ids[i] = getNewId(element.subtopics_ids[i])
        }
    }
}

//////////////////////////

var topicCursor = db.getCollection(collection).find()
while (topicCursor.hasNext()) {
    var topic = topicCursor.next()
    updateTopicDocument(topic)
    if (cfg.verbose) {
        printjson(topic)
    }
    if (cfg.update) {
        db.getCollection(collection).updateOne({_id:topic._id}, {$set : topic} )
    }
}
