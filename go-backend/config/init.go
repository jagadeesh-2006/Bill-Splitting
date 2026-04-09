package config

func init(){
	loadenv()
	ConnectDB()
}