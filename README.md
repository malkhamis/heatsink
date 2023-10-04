# Historical Background
I started this project out of the frustration with the lack of proper CPU thermal management in Dell Desktops with Linux. While I have set it up as a service with systemctl and it has been working without a problem (thanks to all these unit tests), I could never have the time to provide proper documentation, sample configurations, RPM package, etc.. 

# Technical Summary
Because I work on Linux, I only provided interface implementations for the `Sensor` and `FanDriver` that are meant for Linux. These implementations are based on the notion that the means to interact with a device in Linux is through a file. That is, to obtain a thermal reading, we must read from some file; and to control the fan speed, we must write to some file.

