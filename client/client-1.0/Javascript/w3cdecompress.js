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
            finalMsg += ':' //colon
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
                const todayYr = new Date();
                timestamp = parseInt(Math.floor(todayYr.getFullYear()));
                var byte1 = message.charCodeAt(index);
                var byte2 = message.charCodeAt(index+1);
                var byte3 = message.charCodeAt(index+2);
                var byte4 = message.charCodeAt(index+3);

                var yy  = parseInt((byte1 & 0b00111100) >> 2)
                timestamp += yy
                timestamp += '-'
                
                var mm1 = (byte1 & 0b00000011)
                var mm2 = (byte2 & 0b11000000) >> 6
                var mm  = parseInt(mm1 + mm2)
                timestamp += mm
                timestamp += '-'

                var dd  = parseInt((byte2 & 0b00111110) >> 1)
                timestamp += dd                
                timestamp += 'T'

                var hh1 = (byte2 & 0b00000001)
                var hh2 = (byte3 & 0b11110000) >> 4
                var hh  = parseInt(hh1 + hh2)
                timestamp += hh                
                timestamp += ':'

                var MM1 = (byte3 & 0b00001111)
                var MM2 = (byte4 & 0b11000000) >> 6
                var MM  = parseInt(MM1 + MM2)
                timestamp += MM                
                timestamp += ':'

                var ss =  parseInt(byte4 & 0b00111111)
                timestamp += ss                
                timestamp += 'Z'

                finalMsg = finalMsg + '"' + timestamp + '"'
                console.log("Timestamp assigned " + timestamp )
                index = index + 4
            }
        } else {
            finalMsg += message.charAt(index)
            index = index+1
        }
    }
    return '{' + finalMsg + '}'
}
