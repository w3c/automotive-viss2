/**
* (C) 2021 Geotab
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/WAII
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/
package utils

import (
	"encoding/json"
	"strconv"
//	"strings"

//	"github.com/MEAE-GOT/WAII/utils"
	pb "github.com/MEAE-GOT/WAII/protobuf/protoc-out"
	"github.com/golang/protobuf/proto"
//	"github.com/akamensky/argparse"
//	"github.com/gorilla/websocket"
)

func ProtobufToJson(serialisedMessage []byte) string {
    jsonMessage := ""
    deSerialisedMessage := &pb.ProtobufMessage{}
    err := proto.Unmarshal(serialisedMessage, deSerialisedMessage)
    if err != nil {
        Error.Printf("Unmarshaling error: ", err)
        return ""
    }
    // populate jsonMessage from deSerialisedMessage
    return jsonMessage
}

func JsonToProtobuf(jsonMessage string) []byte {
    var protoMessage *pb.ProtobufMessage
    protoMessage = populateProtoFromJson(jsonMessage)
testPrintProtoMessage(protoMessage)
    serialisedMessage, err := proto.Marshal(protoMessage)
    if err != nil {
        Error.Printf("Unmarshaling error: ", err)
        return nil
    }
Info.Printf("JSON size=%d, ProtoBuf size = %d", len(jsonMessage), len(serialisedMessage))
Info.Printf("JSON size / ProtoBuf size=%d%", (100*len(jsonMessage))/len(serialisedMessage))
    return serialisedMessage
}

func populateProtoFromJson(jsonMessage string) *pb.ProtobufMessage {
    protoMessage := &pb.ProtobufMessage{}
    var messageMap map[string]interface{}
    err := json.Unmarshal([]byte(jsonMessage), &messageMap)
    if err != nil {
        Error.Printf("populateProtoFromJson:Unmarshal error data=%s, err=%s", jsonMessage, err)
	return nil
    }
    mMethod, mType := getMethodAndType(messageMap)
    if mMethod == -1 {
        Error.Printf("Unknown message format=%s", jsonMessage)
        return nil
    }
    protoMessage.Method = mMethod
    switch mMethod {
        case pb.MessageMethod_GET:
            createGetPb(protoMessage, messageMap, mType)
        case pb.MessageMethod_SET:
            createSetPb(protoMessage, messageMap, mType)
        case pb.MessageMethod_SUBSCRIBE:
            createSubscribePb(protoMessage, messageMap, mType)
        case pb.MessageMethod_UNSUBSCRIBE:
            createUnSubscribePb(protoMessage, messageMap, mType)
    }
    return protoMessage
}

func getMethodAndType(messageMap map[string]interface{}) (pb.MessageMethod, pb.MessageType) {
    mType := pb.MessageType_REQUEST
    switch messageMap["action"].(string) {
      case "get":
          if messageMap["path"] == nil {
              mType = pb.MessageType_RESPONSE
          }
          return pb.MessageMethod_GET, mType
      case "set":
          if messageMap["path"] == nil {
              mType = pb.MessageType_RESPONSE
          }
          return pb.MessageMethod_SET, mType
      case "subscribe":
          if messageMap["path"] == nil {
              mType = pb.MessageType_RESPONSE
          }
          return pb.MessageMethod_SUBSCRIBE, mType
      case "unsubscribe":
          if messageMap["ts"] != nil {
              mType = pb.MessageType_RESPONSE
          }
          return pb.MessageMethod_UNSUBSCRIBE, mType
      case "subscription":
          return pb.MessageMethod_SUBSCRIBE, pb.MessageType_NOTIFICATION
  }
  return -1, -1
}

func createGetPb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}, mType pb.MessageType) {
    protoMessage.Get = &pb.GetMessage{}
    protoMessage.Get.MType = mType
    switch mType {
        case pb.MessageType_REQUEST: createGetRequestPb(protoMessage, messageMap)
        case pb.MessageType_RESPONSE: createGetResponsePb(protoMessage, messageMap)
    }
}

func createGetRequestPb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}) {
    protoMessage.Get.Request = &pb.GetMessage_RequestMessage{}
    protoMessage.Get.Request.Path = messageMap["path"].(string)
    if messageMap["filter"] != nil {
        filter := messageMap["filter"]
        switch vv := filter.(type) {
          case []interface{}:
            Info.Println(filter, "is an array:, len=",strconv.Itoa(len(vv)))
            if (len(vv) != 2) {
                Error.Printf("Max two filter expressions are allowed.")
                break
            }
            protoMessage.Get.Request.Filter = &pb.FilterExpressions{}
            protoMessage.Get.Request.Filter.FilterExp = make([]*pb.FilterExpressions_FilterExpression, 2)
            protoMessage.Get.Request.Filter.FilterExp[0] = &pb.FilterExpressions_FilterExpression{}
            protoMessage.Get.Request.Filter.FilterExp[1] = &pb.FilterExpressions_FilterExpression{}
            createPbFilter(0, vv[0].(map[string]interface{}), protoMessage)
            createPbFilter(1, vv[1].(map[string]interface{}), protoMessage)
          case map[string]interface{}:
            Info.Println(vv, "is a map:")
            protoMessage.Get.Request.Filter = &pb.FilterExpressions{}
            protoMessage.Get.Request.Filter.FilterExp = make([]*pb.FilterExpressions_FilterExpression, 1)
            protoMessage.Get.Request.Filter.FilterExp[0] = &pb.FilterExpressions_FilterExpression{}
            createPbFilter(0, vv, protoMessage)
          default:
            Info.Println(filter, "is of an unknown type")
        }            
    }
    if messageMap["authorization"] != nil {
        auth := messageMap["authorization"].(string)
        protoMessage.Get.Request.Authorization = &auth
    }
    if messageMap["requestId"] != nil {
        reqId := messageMap["requestId"].(string)
        protoMessage.Get.Request.RequestId = &reqId
    }
}

func createGetResponsePb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}) {
    protoMessage.Get.Response = &pb.GetMessage_ResponseMessage{}
    if (messageMap["error"] == nil) {
        protoMessage.Get.Response.Status = pb.ResponseStatus_SUCCESS
        protoMessage.Get.Response.SuccessResponse = &pb.GetMessage_ResponseMessage_SuccessResponseMessage{}
        protoMessage.Get.Response.SuccessResponse.Action = messageMap["action"].(string)
        numOfDataElements := getNumOfDataElements(messageMap["data"])
        if (numOfDataElements > 0) {
            protoMessage.Get.Response.SuccessResponse.DataPack = &pb.DataPackages{}
            protoMessage.Get.Response.SuccessResponse.DataPack.Data = make([]*pb.DataPackages_DataPackage, numOfDataElements)
            for i := 0 ; i < numOfDataElements ; i++ {
                protoMessage.Get.Response.SuccessResponse.DataPack.Data[i] = createDataElement(i, messageMap["data"])
            }
        } else {
	    metadata, _ := json.Marshal(messageMap["metadata"])
	    metadataStr := string(metadata)
            protoMessage.Get.Response.SuccessResponse.Metadata = &metadataStr
        }
        requestId := messageMap["requestId"].(string)
        protoMessage.Get.Response.SuccessResponse.RequestId = &requestId
        protoMessage.Get.Response.SuccessResponse.Ts = messageMap["ts"].(string)
    } else {
        protoMessage.Get.Response.Status = pb.ResponseStatus_ERROR
        protoMessage.Get.Response.ErrorResponse = &pb.ErrorResponseMessage{}
        protoMessage.Get.Response.ErrorResponse.Action = messageMap["action"].(string)
        protoMessage.Get.Response.ErrorResponse.Err = &pb.ErrorResponseMessage_ErrorMessage{}
        protoMessage.Get.Response.ErrorResponse.Err = getProtoErrorMessage(messageMap["error"])
        requestId := messageMap["requestId"].(string)
        protoMessage.Get.Response.ErrorResponse.RequestId = &requestId
        protoMessage.Get.Response.ErrorResponse.Ts = messageMap["ts"].(string)
    }
}

func getProtoErrorMessage(messageErrorMap interface{}) *pb.ErrorResponseMessage_ErrorMessage {
    var errorObject map[string]interface{}
    switch vv := messageErrorMap.(type) {
      case map[string]interface{}: errorObject = vv
      default: return nil
    }
    var protoErrorMessage pb.ErrorResponseMessage_ErrorMessage
    protoErrorMessage.Number = errorObject["number"].(string)
    if (errorObject["reason"] != nil) {
        reason := errorObject["reason"].(string)
        protoErrorMessage.Reason = &reason
    }
    if (errorObject["reason"] != nil) {
        message := errorObject["message"].(string)
        protoErrorMessage.Message = &message
    }
    return &protoErrorMessage
}

func getNumOfDataElements(messageDataMap interface{}) int {
    if (messageDataMap == nil) {
        return 0
    }
    switch vv := messageDataMap.(type) {
      case []interface{}: return len(vv)
    }
    return 1
}

func createDataElement(index int, messageDataMap interface{}) *pb.DataPackages_DataPackage {
    var dataObject map[string]interface{}
    switch vv := messageDataMap.(type) {
      case []interface{}: dataObject = vv[index].(map[string]interface{})
      default: dataObject = vv.(map[string]interface{})
    }
    var protoDataElement pb.DataPackages_DataPackage
    protoDataElement.Path = dataObject["path"].(string)
    numOfDataPointElements := getNumOfDataPointElements(dataObject["dp"])
    protoDataElement.Dp = make([]*pb.DataPackages_DataPackage_DataPoint, numOfDataPointElements)
    for i := 0 ; i < numOfDataPointElements ; i++ {
        protoDataElement.Dp[i] = createDataPointElement(i, dataObject["dp"])
    }
    return &protoDataElement
}

func getNumOfDataPointElements(messageDataPointMap interface{}) int {
    if (messageDataPointMap == nil) {
        return 0
    }
    switch vv := messageDataPointMap.(type) {
      case []interface{}: return len(vv)
    }
    return 1
}

func createDataPointElement(index int, messageDataPointMap interface{}) *pb.DataPackages_DataPackage_DataPoint {
    var dataPointObject map[string]interface{}
    switch vv := messageDataPointMap.(type) {
      case []interface{}: dataPointObject = vv[index].(map[string]interface{})
      default: dataPointObject = vv.(map[string]interface{})
    }
    var protoDataPointElement pb.DataPackages_DataPackage_DataPoint
    protoDataPointElement.Value = dataPointObject["value"].(string)
    protoDataPointElement.Ts = dataPointObject["ts"].(string)
    return &protoDataPointElement
}

func createPbFilter(index int, filterExpression map[string]interface{}, protoMessage *pb.ProtobufMessage) {
        filterType := getFilterType(filterExpression["type"].(string))
        if (protoMessage.Method == pb.MessageMethod_GET) {
            protoMessage.Get.Request.Filter.FilterExp[index].FType = filterType
        } else {
            protoMessage.Subscribe.Request.Filter.FilterExp[index].FType = filterType
        }
        if (protoMessage.Method == pb.MessageMethod_GET && 
            (filterType == pb.FilterExpressions_FilterExpression_TIMEBASED || 
            filterType == pb.FilterExpressions_FilterExpression_RANGE || 
            filterType == pb.FilterExpressions_FilterExpression_CHANGE || 
            filterType == pb.FilterExpressions_FilterExpression_CURVELOG)) {
                Error.Printf("Filter function is not supported for GET requests.")
                return
        }
        if (protoMessage.Method == pb.MessageMethod_SUBSCRIBE && 
            (filterType == pb.FilterExpressions_FilterExpression_HISTORY || 
            filterType == pb.FilterExpressions_FilterExpression_STATIC_METADATA || 
            filterType == pb.FilterExpressions_FilterExpression_DYNAMIC_METADATA)) {
                Error.Printf("Filter function is not supported for SUBSCRIBE requests.")
                return
        }
        if (protoMessage.Method == pb.MessageMethod_GET) {
            protoMessage.Get.Request.Filter.FilterExp[index].Value = &pb.FilterExpressions_FilterExpression_FilterValue{}
        } else {
            protoMessage.Subscribe.Request.Filter.FilterExp[index].Value = &pb.FilterExpressions_FilterExpression_FilterValue{}
        }
        switch filterType {
            case pb.FilterExpressions_FilterExpression_PATHS:
                if (protoMessage.Method == pb.MessageMethod_GET) {
                    protoMessage.Get.Request.Filter.FilterExp[index].Value.ValuePaths = 
                    &pb.FilterExpressions_FilterExpression_FilterValue_PathsValue{}
                    protoMessage.Get.Request.Filter.FilterExp[index].Value.ValuePaths = getPbPathsFilterValue(filterExpression["value"])
                } else {
                    protoMessage.Subscribe.Request.Filter.FilterExp[index].Value.ValuePaths = 
                    &pb.FilterExpressions_FilterExpression_FilterValue_PathsValue{}
                    protoMessage.Subscribe.Request.Filter.FilterExp[index].Value.ValuePaths = getPbPathsFilterValue(filterExpression["value"])
                }
            case pb.FilterExpressions_FilterExpression_TIMEBASED:
                protoMessage.Subscribe.Request.Filter.FilterExp[index].Value.ValueTimebased = 
                &pb.FilterExpressions_FilterExpression_FilterValue_TimebasedValue{}
                protoMessage.Subscribe.Request.Filter.FilterExp[index].Value.ValueTimebased = 
                getPbTimebasedFilterValue(filterExpression["value"].(map[string]interface{}))
            case pb.FilterExpressions_FilterExpression_RANGE:
                rangeLen := getNumOfRangeExpressions(filterExpression["value"])
                protoMessage.Subscribe.Request.Filter.FilterExp[index].Value.ValueRange = 
                make([]*pb.FilterExpressions_FilterExpression_FilterValue_RangeValue, rangeLen)
                for i := 0 ; i < rangeLen ; i++ {
                    protoMessage.Subscribe.Request.Filter.FilterExp[index].Value.ValueRange[i] = 
                    getPbRangeFilterValue(i, filterExpression["value"])
                }
            case pb.FilterExpressions_FilterExpression_CHANGE:
                protoMessage.Subscribe.Request.Filter.FilterExp[index].Value.ValueChange = 
                &pb.FilterExpressions_FilterExpression_FilterValue_ChangeValue{}
                protoMessage.Subscribe.Request.Filter.FilterExp[index].Value.ValueChange = 
                getPbChangeFilterValue(filterExpression["value"].(map[string]interface{}))
            case pb.FilterExpressions_FilterExpression_CURVELOG:
                protoMessage.Subscribe.Request.Filter.FilterExp[index].Value.ValueCurvelog = 
                &pb.FilterExpressions_FilterExpression_FilterValue_CurvelogValue{}
                protoMessage.Subscribe.Request.Filter.FilterExp[index].Value.ValueCurvelog = 
                getPbCurvelogFilterValue(filterExpression["value"].(map[string]interface{}))
            case pb.FilterExpressions_FilterExpression_HISTORY:
                protoMessage.Get.Request.Filter.FilterExp[index].Value.ValueHistory =
                &pb.FilterExpressions_FilterExpression_FilterValue_HistoryValue{}
                protoMessage.Get.Request.Filter.FilterExp[index].Value.ValueHistory.TimePeriod = filterExpression["value"].(string)
            case pb.FilterExpressions_FilterExpression_STATIC_METADATA:
                Warning.Printf("Filter type is not supported by protobuf compression.")
            case pb.FilterExpressions_FilterExpression_DYNAMIC_METADATA:
                protoMessage.Get.Request.Filter.FilterExp[index].Value.ValueDynamicMetadata =
                &pb.FilterExpressions_FilterExpression_FilterValue_DynamicMetadataValue{}
                protoMessage.Get.Request.Filter.FilterExp[index].Value.ValueDynamicMetadata.MetadataDomain = filterExpression["value"].(string)
            default:
                Error.Printf("Filter type is unknown.")
        }
}

func getNumOfRangeExpressions(valueMap interface{}) int {
    switch vv := valueMap.(type) {
      case []interface{}:
            return len(vv)
      default:
            return 1
    }
}

func getPbPathsFilterValue(filterValueExpression interface{}) *pb.FilterExpressions_FilterExpression_FilterValue_PathsValue {
    var protoPathsValue pb.FilterExpressions_FilterExpression_FilterValue_PathsValue
        switch vv := filterValueExpression.(type) {
          case []interface{}:
            Info.Println(filterValueExpression, "is a string array:, len=",strconv.Itoa(len(vv)))
            protoPathsValue.RelativePath = make([]string,len(vv))
            for i := 0 ; i < len(vv) ; i++ {
                protoPathsValue.RelativePath[i] = vv[i].(string)
            }
          case string:
            Info.Println(filterValueExpression, "is a string:")
            protoPathsValue.RelativePath = make([]string,1)
            protoPathsValue.RelativePath[0] = vv
          default:
            Info.Println(filterValueExpression, "is of an unknown type")
        }
        return &protoPathsValue
}

func getPbTimebasedFilterValue(filterExpression map[string]interface{}) *pb.FilterExpressions_FilterExpression_FilterValue_TimebasedValue {
    var protoTimebasedValue pb.FilterExpressions_FilterExpression_FilterValue_TimebasedValue
    protoTimebasedValue.Period = filterExpression["period"].(string)
    return &protoTimebasedValue
}

func getPbRangeFilterValue(index int, valueMap interface{}) *pb.FilterExpressions_FilterExpression_FilterValue_RangeValue {
    var protoRangeValue pb.FilterExpressions_FilterExpression_FilterValue_RangeValue
    switch vv := valueMap.(type) {
      case []interface{}:
        rangeObject := vv[index].(map[string]interface{})
        protoRangeValue.LogicOperator = rangeObject["logic-op"].(string)
        protoRangeValue.Boundary = rangeObject["boundary"].(string)
      case map[string]interface{}:
        protoRangeValue.LogicOperator = vv["logic-op"].(string)
        protoRangeValue.Boundary = vv["boundary"].(string)
      default: return nil
    }
    return &protoRangeValue
}

func getPbChangeFilterValue(filterExpression map[string]interface{}) *pb.FilterExpressions_FilterExpression_FilterValue_ChangeValue {
    var protoChangeValue pb.FilterExpressions_FilterExpression_FilterValue_ChangeValue
    protoChangeValue.LogicOperator = filterExpression["logic-op"].(string)
    protoChangeValue.Diff = filterExpression["diff"].(string)
    return &protoChangeValue
}

func getPbCurvelogFilterValue(filterExpression map[string]interface{}) *pb.FilterExpressions_FilterExpression_FilterValue_CurvelogValue {
    var protoCurvelogValue pb.FilterExpressions_FilterExpression_FilterValue_CurvelogValue
    protoCurvelogValue.MaxErr = filterExpression["maxerr"].(string)
    protoCurvelogValue.BufSize = filterExpression["bufsize"].(string)
    return &protoCurvelogValue
}

func getFilterType(filterType string) pb.FilterExpressions_FilterExpression_FilterType {
    switch filterType {
        case "paths": return pb.FilterExpressions_FilterExpression_PATHS
        case "timebased": return pb.FilterExpressions_FilterExpression_TIMEBASED
        case "range": return pb.FilterExpressions_FilterExpression_RANGE
        case "change": return pb.FilterExpressions_FilterExpression_CHANGE
        case "curvelog": return pb.FilterExpressions_FilterExpression_CURVELOG
        case "history": return pb.FilterExpressions_FilterExpression_HISTORY
        case "static-metadata": return pb.FilterExpressions_FilterExpression_STATIC_METADATA
        case "dynamic-metadata": return pb.FilterExpressions_FilterExpression_DYNAMIC_METADATA
    }
    return pb.FilterExpressions_FilterExpression_DYNAMIC_METADATA + 100  //undefined filter type
}

func createSubscribePb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}, mType pb.MessageType) {
    protoMessage.Subscribe = &pb.SubscribeMessage{}
    protoMessage.Subscribe.MType = mType
    switch mType {
        case pb.MessageType_REQUEST: createSubscribeRequestPb(protoMessage, messageMap)
        case pb.MessageType_RESPONSE: createSubscribeResponsePb(protoMessage, messageMap)
        case pb.MessageType_NOTIFICATION: createSubscribeNotificationPb(protoMessage, messageMap)
    }
}

func createSubscribeRequestPb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}) {
    protoMessage.Subscribe = &pb.SubscribeMessage{}
    protoMessage.Subscribe.Request = &pb.SubscribeMessage_RequestMessage{}
    protoMessage.Subscribe.Request.Path = messageMap["path"].(string)
    if messageMap["filter"] != nil {
        filter := messageMap["filter"]
        switch vv := filter.(type) {
          case []interface{}:
            Info.Println(filter, "is an array:, len=",strconv.Itoa(len(vv)))
            if (len(vv) != 2) {
                Error.Printf("Max two filter expressions are allowed.")
                break
            }
            protoMessage.Subscribe.Request.Filter = &pb.FilterExpressions{}
            protoMessage.Subscribe.Request.Filter.FilterExp = make([]*pb.FilterExpressions_FilterExpression, 2)
            protoMessage.Subscribe.Request.Filter.FilterExp[0] = &pb.FilterExpressions_FilterExpression{}
            protoMessage.Subscribe.Request.Filter.FilterExp[1] = &pb.FilterExpressions_FilterExpression{}
            createPbFilter(0, vv[0].(map[string]interface{}), protoMessage)
            createPbFilter(1, vv[1].(map[string]interface{}), protoMessage)
          case map[string]interface{}:
            Info.Println(filter, "is a map:")
            protoMessage.Subscribe.Request.Filter = &pb.FilterExpressions{}
            protoMessage.Subscribe.Request.Filter.FilterExp = make([]*pb.FilterExpressions_FilterExpression, 1)
            protoMessage.Subscribe.Request.Filter.FilterExp[0] = &pb.FilterExpressions_FilterExpression{}
            createPbFilter(0, vv, protoMessage)
          default:
            Info.Println(filter, "is of an unknown type")
        }            
    }
    if messageMap["authorization"] != nil {
        auth := messageMap["authorization"].(string)
        protoMessage.Subscribe.Request.Authorization = &auth
    }
    if messageMap["requestId"] != nil {
        reqId := messageMap["requestId"].(string)
        protoMessage.Subscribe.Request.RequestId = reqId
    }
}

func createSubscribeResponsePb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}) {
    protoMessage.Subscribe.Response = &pb.SubscribeMessage_ResponseMessage{}
    if (messageMap["error"] == nil) {
        protoMessage.Subscribe.Response.Status = pb.ResponseStatus_SUCCESS
        protoMessage.Subscribe.Response.SuccessResponse = &pb.SubscribeMessage_ResponseMessage_SuccessResponseMessage{}
        protoMessage.Subscribe.Response.SuccessResponse.Action = messageMap["action"].(string)
        protoMessage.Subscribe.Response.SuccessResponse.SubscriptionId = messageMap["subscriptionId"].(string)
        protoMessage.Subscribe.Response.SuccessResponse.Ts = messageMap["ts"].(string)
    } else {
        protoMessage.Subscribe.Response.Status = pb.ResponseStatus_ERROR
        protoMessage.Subscribe.Response.ErrorResponse = &pb.ErrorResponseMessage{}
        protoMessage.Subscribe.Response.ErrorResponse.Action = messageMap["action"].(string)
        protoMessage.Subscribe.Response.ErrorResponse.Err = &pb.ErrorResponseMessage_ErrorMessage{}
        protoMessage.Subscribe.Response.ErrorResponse.Err = getProtoErrorMessage(messageMap["error"])
        requestId := messageMap["requestId"].(string)
        protoMessage.Subscribe.Response.ErrorResponse.RequestId = &requestId
        protoMessage.Subscribe.Response.ErrorResponse.Ts = messageMap["ts"].(string)
    }
}

func createSubscribeNotificationPb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}) {
    protoMessage.Subscribe.Notification = &pb.SubscribeMessage_NotificationMessage{}
    if (messageMap["error"] == nil) {
        protoMessage.Subscribe.Notification.Status = pb.ResponseStatus_SUCCESS
        protoMessage.Subscribe.Notification.SuccessResponse = &pb.SubscribeMessage_NotificationMessage_SuccessResponseMessage{}
        protoMessage.Subscribe.Notification.SuccessResponse.Action = messageMap["action"].(string)
        numOfDataElements := getNumOfDataElements(messageMap["data"])
        protoMessage.Subscribe.Notification.SuccessResponse.DataPack = &pb.DataPackages{}
        protoMessage.Subscribe.Notification.SuccessResponse.DataPack.Data = make([]*pb.DataPackages_DataPackage, numOfDataElements)
        for i := 0 ; i < numOfDataElements ; i++ {
            protoMessage.Subscribe.Notification.SuccessResponse.DataPack.Data[i] = createDataElement(i, messageMap["data"])
        }
        protoMessage.Subscribe.Notification.SuccessResponse.SubscriptionId = messageMap["subscriptionId"].(string)
        protoMessage.Subscribe.Notification.SuccessResponse.Ts = messageMap["ts"].(string)
    } else {
        protoMessage.Subscribe.Notification.Status = pb.ResponseStatus_ERROR
        protoMessage.Subscribe.Notification.ErrorResponse = &pb.ErrorResponseMessage{}
        protoMessage.Subscribe.Notification.ErrorResponse.Action = messageMap["action"].(string)
        protoMessage.Subscribe.Notification.ErrorResponse.Err = &pb.ErrorResponseMessage_ErrorMessage{}
        protoMessage.Subscribe.Notification.ErrorResponse.Err = getProtoErrorMessage(messageMap["error"])
        subscriptionId := messageMap["subscriptionId"].(string)
        protoMessage.Subscribe.Notification.ErrorResponse.SubscriptionId = &subscriptionId
        protoMessage.Subscribe.Notification.ErrorResponse.Ts = messageMap["ts"].(string)
    }
}

func createSetPb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}, mType pb.MessageType) {
    protoMessage.Set = &pb.SetMessage{}
    protoMessage.Set.MType = mType
    switch mType {
        case pb.MessageType_REQUEST: createSetRequestPb(protoMessage, messageMap)
        case pb.MessageType_RESPONSE: createSetResponsePb(protoMessage, messageMap)
    }
}

func createSetRequestPb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}) {
    protoMessage.Set.Request = &pb.SetMessage_RequestMessage{}
    protoMessage.Set.Request.Path = messageMap["path"].(string)
    protoMessage.Set.Request.Value = messageMap["value"].(string)
    if messageMap["authorization"] != nil {
        auth := messageMap["authorization"].(string)
        protoMessage.Set.Request.Authorization = &auth
    }
    if messageMap["requestId"] != nil {
        reqId := messageMap["requestId"].(string)
        protoMessage.Set.Request.RequestId = &reqId
    }
}

func createSetResponsePb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}) {
    protoMessage.Set.Response = &pb.SetMessage_ResponseMessage{}
    if (messageMap["error"] == nil) {
        protoMessage.Set.Response.Status = pb.ResponseStatus_SUCCESS
        protoMessage.Set.Response.SuccessResponse = &pb.SetMessage_ResponseMessage_SuccessResponseMessage{}
        protoMessage.Set.Response.SuccessResponse.Action = messageMap["action"].(string)
        requestId := messageMap["requestId"].(string)
        protoMessage.Set.Response.SuccessResponse.RequestId = &requestId
        protoMessage.Set.Response.SuccessResponse.Ts = messageMap["ts"].(string)
    } else {
        protoMessage.Set.Response.Status = pb.ResponseStatus_ERROR
        protoMessage.Set.Response.ErrorResponse = &pb.ErrorResponseMessage{}
        protoMessage.Set.Response.ErrorResponse.Action = messageMap["action"].(string)
        protoMessage.Set.Response.ErrorResponse.Err = &pb.ErrorResponseMessage_ErrorMessage{}
        protoMessage.Set.Response.ErrorResponse.Err = getProtoErrorMessage(messageMap["error"])
        requestId := messageMap["requestId"].(string)
        protoMessage.Set.Response.ErrorResponse.RequestId = &requestId
        protoMessage.Set.Response.ErrorResponse.Ts = messageMap["ts"].(string)
    }
}

func createUnSubscribePb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}, mType pb.MessageType) {
    protoMessage.UnSubscribe = &pb.UnSubscribeMessage{}
    protoMessage.UnSubscribe.MType = mType
    switch mType {
        case pb.MessageType_REQUEST: createUnSubscribeRequestPb(protoMessage, messageMap)
        case pb.MessageType_RESPONSE: createUnSubscribeResponsePb(protoMessage, messageMap)
    }
}

func createUnSubscribeRequestPb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}) {
    protoMessage.UnSubscribe.Request = &pb.UnSubscribeMessage_RequestMessage{}
    protoMessage.UnSubscribe.Request.SubscriptionId = messageMap["subscriptionId"].(string)
    if messageMap["requestId"] != nil {
        reqId := messageMap["requestId"].(string)
        protoMessage.UnSubscribe.Request.RequestId = &reqId
    }
}

func createUnSubscribeResponsePb(protoMessage *pb.ProtobufMessage, messageMap map[string]interface{}) {
    protoMessage.UnSubscribe.Response = &pb.UnSubscribeMessage_ResponseMessage{}
    if (messageMap["error"] == nil) {
        protoMessage.UnSubscribe.Response.Status = pb.ResponseStatus_SUCCESS
        protoMessage.UnSubscribe.Response.SuccessResponse = &pb.UnSubscribeMessage_ResponseMessage_SuccessResponseMessage{}
        protoMessage.UnSubscribe.Response.SuccessResponse.Action = messageMap["action"].(string)
        protoMessage.UnSubscribe.Response.SuccessResponse.SubscriptionId = messageMap["subscriptionId"].(string)
        protoMessage.UnSubscribe.Response.SuccessResponse.Ts = messageMap["ts"].(string)
    } else {
        protoMessage.UnSubscribe.Response.Status = pb.ResponseStatus_ERROR
        protoMessage.UnSubscribe.Response.ErrorResponse = &pb.ErrorResponseMessage{}
        protoMessage.UnSubscribe.Response.ErrorResponse.Action = messageMap["action"].(string)
        protoMessage.UnSubscribe.Response.ErrorResponse.Err = &pb.ErrorResponseMessage_ErrorMessage{}
        protoMessage.UnSubscribe.Response.ErrorResponse.Err = getProtoErrorMessage(messageMap["error"])
        subscriptionId := messageMap["subscriptionId"].(string)
        protoMessage.UnSubscribe.Response.ErrorResponse.SubscriptionId = &subscriptionId
        protoMessage.UnSubscribe.Response.ErrorResponse.Ts = messageMap["ts"].(string)
    }
}

//      *******************************Only for testing during dev ***************************************
func testPrintProtoMessage(protoMessage *pb.ProtobufMessage) {
  switch protoMessage.GetMethod() {
    case 0: // GET
      switch protoMessage.GetGet().GetMType() {
        case 0: //REQUEST
          Info.Printf("protoMessage.Method = %d, protoMessage.Get.MType=%d", protoMessage.GetMethod(), protoMessage.GetGet().GetMType())
          Info.Printf("protoMessage.Get.Request.Path = %s", protoMessage.GetGet().GetRequest().GetPath())
          Info.Printf("protoMessage.Get.Request.RequestId = %s", protoMessage.GetGet().GetRequest().GetRequestId())
        case 1: // RESPONSE
          Info.Printf("protoMessage.Method = %d, protoMessage.Get.MType=%d", protoMessage.GetMethod(), protoMessage.GetGet().GetMType())
          if (protoMessage.GetGet().GetResponse().GetStatus() == 0) { //SUCCESSFUL
            Info.Printf("protoMessage.Get.Response.Status=%d", protoMessage.GetGet().GetResponse().GetStatus())
            Info.Printf("protoMessage.Get.Response.RequestId = %s", protoMessage.GetGet().GetResponse().GetSuccessResponse().GetRequestId())
            Info.Printf("protoMessage.Get.Response.Ts = %s", protoMessage.GetGet().GetResponse().GetSuccessResponse().GetTs())
          } else { // ERROR
            Info.Printf("protoMessage.Get.Response.Status=%d", protoMessage.GetGet().GetResponse().GetStatus())
            Info.Printf("protoMessage.Get.Response.RequestId = %s", protoMessage.GetGet().GetResponse().GetErrorResponse().GetRequestId())
            Info.Printf("protoMessage.Get.Response.Ts = %s", protoMessage.GetGet().GetResponse().GetErrorResponse().GetTs())
          }
      }
    case 1: // SET
      switch protoMessage.GetSet().GetMType() {
        case 0: //REQUEST
          Info.Printf("protoMessage.Method = %d, protoMessage.Get.MType=%d", protoMessage.GetMethod(), protoMessage.GetSet().GetMType())
          Info.Printf("protoMessage.Set.Request.Path = %s", protoMessage.GetSet().GetRequest().GetPath())
          Info.Printf("protoMessage.Set.Request.RequestId = %s", protoMessage.GetSet().GetRequest().GetRequestId())
        case 1: // RESPONSE
          Info.Printf("protoMessage.Method = %d, protoMessage.Get.MType=%d", protoMessage.GetMethod(), protoMessage.GetSet().GetMType())
          if (protoMessage.GetSet().GetResponse().GetStatus() == 0) { //SUCCESSFUL
            Info.Printf("protoMessage.Set.Response.Status=%d", protoMessage.GetSet().GetResponse().GetStatus())
            Info.Printf("protoMessage.Set.Response.RequestId = %s", protoMessage.GetSet().GetResponse().GetSuccessResponse().GetRequestId())
            Info.Printf("protoMessage.Set.Response.Ts = %s", protoMessage.GetSet().GetResponse().GetSuccessResponse().GetTs())
          } else { // ERROR
            Info.Printf("protoMessage.Set.Response.Status=%d", protoMessage.GetSet().GetResponse().GetStatus())
            Info.Printf("protoMessage.Set.Response.RequestId = %s", protoMessage.GetSet().GetResponse().GetErrorResponse().GetRequestId())
            Info.Printf("protoMessage.Set.Response.Ts = %s", protoMessage.GetSet().GetResponse().GetErrorResponse().GetTs())
          }
      }
    case 2: // SUBSCRIBE
      switch protoMessage.GetSubscribe().GetMType() {
        case 0: //REQUEST
          Info.Printf("protoMessage.Method = %d, protoMessage.Get.MType=%d", protoMessage.GetMethod(), protoMessage.GetSubscribe().GetMType())
          Info.Printf("protoMessage.Get.Request.Path = %s", protoMessage.GetSubscribe().GetRequest().GetPath())
          Info.Printf("protoMessage.Get.Request.RequestId = %s", protoMessage.GetSubscribe().GetRequest().GetRequestId())
        case 1: // RESPONSE
          Info.Printf("protoMessage.Method = %d, protoMessage.Get.MType=%d", protoMessage.GetMethod(), protoMessage.GetSubscribe().GetMType())
          if (protoMessage.GetSubscribe().GetResponse().GetStatus() == 0) { //SUCCESSFUL
            Info.Printf("protoMessage.Get.Response.Status=%d", protoMessage.GetSubscribe().GetResponse().GetStatus())
            Info.Printf("protoMessage.Get.Response.SubscriptionId = %s", protoMessage.GetSubscribe().GetResponse().GetSuccessResponse().GetSubscriptionId())
            Info.Printf("protoMessage.Get.Response.Ts = %s", protoMessage.GetSubscribe().GetResponse().GetSuccessResponse().GetTs())
          } else { // ERROR
            Info.Printf("protoMessage.Get.Response.Status=%d", protoMessage.GetSubscribe().GetResponse().GetStatus())
            Info.Printf("protoMessage.Get.Response.RequestId = %s", protoMessage.GetSubscribe().GetResponse().GetErrorResponse().GetRequestId())
            Info.Printf("protoMessage.Get.Response.Ts = %s", protoMessage.GetSubscribe().GetResponse().GetErrorResponse().GetTs())
          }
        case 2: // NOTIFICATION
          Info.Printf("protoMessage.Method = %d, protoMessage.Get.MType=%d", protoMessage.GetMethod(), protoMessage.GetSubscribe().GetMType())
          if (protoMessage.GetSubscribe().GetResponse().GetStatus() == 0) { //SUCCESSFUL
            Info.Printf("protoMessage.Get.Notification.Status=%d", protoMessage.GetSubscribe().GetNotification().GetStatus())
            Info.Printf("protoMessage.Get.Notification.SubscriptionId = %s", protoMessage.GetSubscribe().GetNotification().GetSuccessResponse().GetSubscriptionId())
            Info.Printf("protoMessage.Get.Notification.Ts = %s", protoMessage.GetSubscribe().GetNotification().GetSuccessResponse().GetTs())
          } else { // ERROR
            Info.Printf("protoMessage.Get.Notification.Status=%d", protoMessage.GetSubscribe().GetNotification().GetStatus())
            Info.Printf("protoMessage.Get.Notification.SubscriptionId = %s", protoMessage.GetSubscribe().GetNotification().GetErrorResponse().GetSubscriptionId())
            Info.Printf("protoMessage.Get.Notification.Ts = %s", protoMessage.GetSubscribe().GetNotification().GetErrorResponse().GetTs())
          }
      }
    case 3: // UNSUBSCRIBE
      switch protoMessage.GetUnSubscribe().GetMType() {
        case 0: //REQUEST
          Info.Printf("protoMessage.Method = %d, protoMessage.Get.MType=%d", protoMessage.GetMethod(), protoMessage.GetUnSubscribe().GetMType())
          Info.Printf("protoMessage.UnSubscribe.Request.SubscriptionId = %s", protoMessage.GetUnSubscribe().GetRequest().GetSubscriptionId())
        case 1: // RESPONSE
          Info.Printf("protoMessage.Method = %d, protoMessage.Get.MType=%d", protoMessage.GetMethod(), protoMessage.GetUnSubscribe().GetMType())
          if (protoMessage.GetUnSubscribe().GetResponse().GetStatus() == 0) { //SUCCESSFUL
            Info.Printf("protoMessage.UnSubscribe.Response.Status=%d", protoMessage.GetUnSubscribe().GetResponse().GetStatus())
            Info.Printf("protoMessage.UnSubscribe.Response.SubscriptionId = %s", protoMessage.GetUnSubscribe().GetResponse().GetSuccessResponse().GetSubscriptionId())
            Info.Printf("protoMessage.UnSubscribe.Response.Ts = %s", protoMessage.GetUnSubscribe().GetResponse().GetSuccessResponse().GetTs())
          } else { // ERROR
            Info.Printf("protoMessage.UnSubscribe.Response.Status=%d", protoMessage.GetUnSubscribe().GetResponse().GetStatus())
            Info.Printf("protoMessage.UnSubscribe.Response.SubscriptionId = %s", protoMessage.GetUnSubscribe().GetResponse().GetErrorResponse().GetSubscriptionId())
            Info.Printf("protoMessage.UnSubscribe.Response.Ts = %s", protoMessage.GetUnSubscribe().GetResponse().GetErrorResponse().GetTs())
          }
      }
  }
}

