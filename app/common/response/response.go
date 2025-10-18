package response

type Response struct {
	StatusCode 	int 			`json:"code"`
	StatusMsg 	string 			`json:"msg"`
}

type ResponseWithData struct {
	StatusCode 	int 			`json:"code"`
	StatusMsg 	string 			`json:"msg"`
	Data 		interface{} 	`json:"data"`
}


func NewResponse(statusCode int, statusMsg string) Response {
	return Response{
		StatusCode: statusCode,
		StatusMsg: statusMsg,
	}
}

func NewResponseWithData(statusCode int, statusMsg string, data interface{}) ResponseWithData {
	return ResponseWithData{
		StatusCode: statusCode,
		StatusMsg: statusMsg,
		Data: data,
	}
}


