

## 项目介绍
golang 7仔大




nohup ./game/game &

gf build -a amd64 -s linux -name game

npm run build:prod

npx cap sync

#启动game
nohup ./game/game &

# 安装 Capacitor 核心和 CLI
npm install --save-dev @capacitor/core @capacitor/cli

# 安装 Android 和 iOS 平台支持
npm install --save-dev @capacitor/android @capacitor/ios

npx cap init

npx cap add android

npm run build

npx cap sync


npx cap open android

./gradlew build --info

{
  "appId": "com.yourcompany.myvueapp",
  "appName": "MyVueApp",
  "webDir": "dist",
  "bundledWebRuntime": false,
  "server": {
    "url": "http://192.168.1.100:5173",
    "cleartext": true
  },
  "plugins": {
    "SplashScreen": {
      "launchShowDuration": 3000,
      "launchAutoHide": true,
      "backgroundColor": "#ffffffff",
      "androidSplashResourceName": "splash",
      "androidScaleType": "CENTER_CROP",
      "showSpinner": false
    }
  },
  "android": {
    "allowMixedContent": true
  }
}

distributionBase=GRADLE_USER_HOME
distributionPath=wrapper/dists
distributionUrl=https://mirrors.cloud.tencent.com/gradle/gradle-8.11.1-all.zip
networkTimeout=1000000
validateDistributionUrl=true
zipStoreBase=GRADLE_USER_HOME
zipStorePath=wrapper/dists

