function stringToBinary(input) {
      const binary = input.toString(2)
      const pad = Math.max(8 - binary.length, 0);
      return '0'.repeat(pad) + binary;
}

function binaryToString(input) {
    let bytesLeft = input;
    let result = '';
  
    while (bytesLeft.length) {
      const byte = bytesLeft.substr(0, 8);
      bytesLeft = bytesLeft.substr(8);  
      result += String.fromCharCode(parseInt(byte, 2));
    }
    return result;
}
  
function decompressMessage(message) {
    var finalMsg = ""
    var actionval = ""
    index = 0
    while (index < message.length) {
        charmsg = message.charCodeAt(index)
        // console.log(charmsg);
        if (charmsg > 127) {
            finalMsg = finalMsg + '"' +  keywordlist["keywords"][charmsg-128] + '"'
            // console.log(keywordlist["keywords"][charmsg-128])
            index = index + 1

            var testmsg = keywordlist["keywords"][charmsg-128]

            if (testmsg == "get" || 
                testmsg == "set" || 
                testmsg == "subscribe" || 
                testmsg == "unsubscribe") {
                    console.log("skiping colon")
            }else{
                finalMsg += ':' //colon
            }

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
                const todayYr = new Date()
                timestamp = parseInt(Math.floor(todayYr.getFullYear()/10)*10)
                
                charmsg = message.charCodeAt(index)
                var byte1 = charmsg
                console.log("byte1 " + byte1 )
                charmsg = message.charCodeAt(index+1)
                var byte2 = charmsg
                console.log("byte2 " + byte2 )
                charmsg = message.charCodeAt(index+2)
                var byte3 = charmsg
                console.log("byte3 " + byte3 )
                charmsg = message.charCodeAt(index+3)
                var byte4 = charmsg
                console.log("byte4 " + byte4 )


                var yy  = parseInt((byte1 & 0b00111100) >>> 2)
                timestamp += parseInt(yy)
                timestamp += '-'
                console.log("Timestamp assigned " + timestamp )
                                
                var mm = ((byte1 & 0b00000011)<<2) | ((byte2 & 0b11000000) >>> 6)
                timestamp += mm
                timestamp += '-'
                console.log("Timestamp assigned " + timestamp )

                var dd  = parseInt((byte2 & 0b00111110) >>> 1)
                timestamp += parseInt(dd)
                timestamp += 'T'
                console.log("Timestamp assigned " + timestamp )

                var hh = ((byte2 & 0b00000001)<<4) | ((byte3 & 0b11110000) >>> 4)
                console.log("Hour  " + hh )
                timestamp += parseInt(hh)                
                timestamp += ':'
                console.log("Timestamp assigned " + timestamp )
                
                var MM = ((byte3 & 0b00001111)<<2) | ((byte4 & 0b11000000) >>> 6)
                console.log("bit1  " + ((byte3 & 0b00001111) <<  2) )
                console.log("bit2  " + ((byte4 & 0b11000000) >>> 6) )
                timestamp += parseInt(MM)                
                timestamp += ':'
                console.log("Timestamp assigned " + timestamp )
                
                var ss =  parseInt(byte4 & 0b00111111)
                timestamp += parseInt(ss)
                timestamp += 'Z'
                console.log("Timestamp assigned " + timestamp )

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
