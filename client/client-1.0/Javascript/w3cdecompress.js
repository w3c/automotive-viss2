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
        console.log("message[" + index + "] " + charmsg);
        if (charmsg > 127) {
            var testmsg = keywordlist["keywords"][charmsg-128]
            index = index + 1
            //keywords
            if (charmsg-128 > 7 && charmsg-128 < 13) {
                    
                    finalMsg = finalMsg + '"' +  keywordlist["keywords"][charmsg-128] + '"'
                    //console.log("skiping colon")
            }else if (charmsg - 128 == 4) { 
//path
                for (var i = 0; i < 4; i++) {
                    actionval[i] = message.charAt(index+i);
                    console.log("av = " + actionval[i])
                }

                console.log("ActionVal = " + actionval)
                for (const [key, value] of uuidmap.entries()) {
                    console.log("Key = " + key + " Value =" + value)
                    if(key.startsWith(actionval)) {
                        finalMsg += '"' + value + '"'
                        break;
                    }
                }
                index = index + 4
            } else if (charmsg-128 > 12 && charmsg-128 < 22){
                var numvals = 0
                finalMsg += '"'
                if(testmsg.startsWith("n")) {
                    finalMsg += '-'
                }
                if(testmsg.endsWith("int8")) {
                    numvals = 1
                }
                if(testmsg.endsWith("int16")) {
                    numvals = 2
                }
                if(testmsg.endsWith("int24")) {
                    numvals = 3
                }
                if(testmsg.endsWith("int32")) {
                    numvals = 4
                }
                if(testmsg.endsWith("bool")) {
                    numvals = 1
                }
                if(numvals!=0){
                    var realval = 0;
                    for(var i = 0; i < numvals ; i++)
                    {
                        realval = (realval << 8) + message.charCodeAt(index + i)
                    }
                    index = index + numvals
                    if (testmsg.endsWith("bool")){
                        if(realval==0){
                            finalMsg += "false"
                        }else{
                            finalMsg += "true"
                        }
                    }else{
                        finalMsg += realval.toString()
                    }
                    numvals = 0
                }else{
                    var i = 0
                    while (message.charCodeAt(index + i)) {
                        console.log(message.charCodeAt(index + i))
                        i = i + 1
                    }
                    index = index + i
                }
                finalMsg += '"'
                continue;
            }else{
                if (charmsg-128 != 22) {
                    finalMsg = finalMsg + '"' +  keywordlist["keywords"][charmsg-128] + '"'
                    console.log("Adding " + keywordlist["keywords"][charmsg-128])
                    finalMsg += ':' //colon
                }
            }

            if (charmsg - 128 == 3) {
//timestamp                
                const todayYr = new Date()
                timestamp = parseInt(Math.floor(todayYr.getFullYear()/10)*10)
                
                charmsg = message.charCodeAt(index)
                var byte1 = charmsg
                // console.log("byte1 " + byte1 )
                charmsg = message.charCodeAt(index+1)
                var byte2 = charmsg
                // console.log("byte2 " + byte2 )
                charmsg = message.charCodeAt(index+2)
                var byte3 = charmsg
                // console.log("byte3 " + byte3 )
                charmsg = message.charCodeAt(index+3)
                var byte4 = charmsg
                // console.log("byte4 " + byte4 )

                var yy  = parseInt((byte1 & 0b00111100) >>> 2)
                timestamp += parseInt(yy)
                timestamp += '-'
                // console.log("Timestamp assigned " + timestamp )
                                
                var mm = ((byte1 & 0b00000011)<<2) | ((byte2 & 0b11000000) >>> 6)
                timestamp += mm.toString().padStart(2, '0')
                timestamp += '-'
                // console.log("Timestamp assigned " + timestamp )

                var dd  = parseInt((byte2 & 0b00111110) >>> 1)
                timestamp += parseInt(dd).toString().padStart(2, '0')
                timestamp += 'T'
                // console.log("Timestamp assigned " + timestamp )

                var hh = ((byte2 & 0b00000001)<<4) | ((byte3 & 0b11110000) >>> 4)
                // console.log("Hour  " + hh )
                timestamp += parseInt(hh).toString().padStart(2, '0')
                timestamp += ':'
                // console.log("Timestamp assigned " + timestamp )
                
                var MM = ((byte3 & 0b00001111)<<2) | ((byte4 & 0b11000000) >>> 6)
                // console.log("bit1  " + ((byte3 & 0b00001111) <<  2) )
                // console.log("bit2  " + ((byte4 & 0b11000000) >>> 6) )
                timestamp += parseInt(MM).toString().padStart(2, '0')
                timestamp += ':'
                // console.log("Timestamp assigned " + timestamp )
                
                var ss =  parseInt(byte4 & 0b00111111)
                timestamp += parseInt(ss).toString().padStart(2, '0')
                timestamp += 'Z'
                // console.log("Timestamp assigned " + timestamp )

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
