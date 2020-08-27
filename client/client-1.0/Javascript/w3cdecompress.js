function decompressMessage(message) {
    var finalMsg = ""
    var actionval = ""
    index = 0
    while (index < message.length) {
        charmsg = message.charCodeAt(index)
        // console.log(charmsg);
        if (charmsg > 127) {
            finalMsg = finalMsg + '"' +  keywordlist["keywords"][charmsg-128] + '"'
            index = index + 1
            finalMsg += message.charAt(index) //colon
            index = index + 1
            // console.log("case " + keywordlist["keywords"][charmsg-128])
            // console.log("case " + charmsg-128)
            if (charmsg - 128 == 1) {
                for (var i = 0; i < 4; i++)
                    actionval[i] = message.charAt(index+i);

                // console.log("ActionVal = " + actionval)
                for (const [key, value] of uuidmap.entries()) {
                    console.log("Key = " + key + " Value =" + value)
                    if(key.startsWith(actionval)) {
                        finalMsg += '"' + value + '"'
                        break;
                    }
                }
                index = index + 4
            } else if (charmsg - 128 == 3) {
                timestamp = "20"
                for (var i=0; i<6; i++) {
                    timestamp += ("00" + message.charCodeAt(index + i)).slice(-2)
                    if (i == 0) timestamp += '-'
                    if (i == 1) timestamp += '-'
                    if (i == 2) timestamp += 'T'
                    if (i == 3) timestamp += ':'
                    if (i == 4) timestamp += ':'
                    if (i == 5) timestamp += 'Z'
                }
                finalMsg = finalMsg + '"' + timestamp + '"'
                // console.log("Timestamp assigned " + timestamp )
                index = index + 6
            }
        } else {
            finalMsg += message.charAt(index)
            index = index+1
        }
    }
    return finalMsg
}
