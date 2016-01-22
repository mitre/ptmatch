/*
Copyright 2016 The MITRE Corporation. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

		http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package logger

import (
	"github.com/Sirupsen/logrus"
)

var Log = logrus.WithFields(logrus.Fields{"app": "popHealth"})

func init() {
	//logrus.SetOutput(os.Stdout)

	//logrus.SetFormatter(&logrus.JSONFormatter{})
	//logrus.SetFormatter(&logrus.TextFormatter{DisableTimestamp: false, DisableColors: true})
	logrus.SetFormatter(&logrus.TextFormatter{})

	//Log only the specified level and more severe
	logrus.SetLevel(logrus.InfoLevel)

	Log.Info("Initializing Logger")
}
