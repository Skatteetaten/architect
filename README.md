```
_______  _______  _______          __________________ _______  _______ _________
(  ___  )(  ____ )(  ____ \|\     /|\__   __/\__   __/(  ____ \(  ____ \\__   __/
| (   ) || (    )|| (    \/| )   ( |   ) (      ) (   | (    \/| (    \/   ) (   
| (___) || (____)|| |      | (___) |   | |      | |   | (__    | |         | |   
|  ___  ||     __)| |      |  ___  |   | |      | |   |  __)   | |         | |   
| (   ) || (\ (   | |      | (   ) |   | |      | |   | (      | |         | |   
| )   ( || ) \ \__| (____/\| )   ( |___) (___   | |   | (____/\| (____/\   | |   
|/     \||/   \__/(_______/|/     \|\_______/   )_(   (_______/(_______/   )_(   
                                                                                 
```

# What is it?

Architect is a docker image that builds other docker image using Openshift CustomBuilder strategy. 

# How to use it?

TODO

# How to build?

```
make # Build the application. Is is written to bin/<achitecture>
make test # Runs test, go vet and go fmt. Should be run before every checkin
```

# Dependecies

We use glide. When you need to install dependencies, use

```
glide install
```

For update of dependecies, see Glide documentation (http://glide.sh)

