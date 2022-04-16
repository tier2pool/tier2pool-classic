package cache

//var (
//	client *redis.Client
//)
//
//func Initialize() error {
//	client = redis.NewClient(&redis.Options{
//		Addr:     "127.0.0.1:6379",
//		Password: "11a6e0ba-1b89-42fc-91ce-c7a7c129a58a",
//	})
//
//	if err := client.Ping(context.Background()).Err(); err != nil {
//		return err
//	}
//
//	logrus.Infoln("connected to redis")
//
//	return nil
//}
