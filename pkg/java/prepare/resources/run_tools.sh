function create_splunk_stanza()
{
    SPLUNK_INDEX="$1"
    OUTPUT_FILE="$2"
    SPLUNK_STANZA="$3"

    SPLUNK_AUDIT_INDEX="audit-test"

    if [ -n "$SPLUNK_INDEX" ]; then

        if [[ $SPLUNK_INDEX =~ .*-prod ]]; then
          SPLUNK_AUDIT_INDEX="audit-prod"
        fi

        if [ -z "$SPLUNK_STANZA"  ]; then
            # In Openshift the host name is the same as pod name. The pod name, by convention, is the application name followed
            # by a dash, then the deployment index followed by a dash, then finally a 5 letter random pod identifier.
            # Example: aurora-openshift-console-91-zzzga
            # We can therefore derive the application name from the host name by removing the deployment index and the pod
            # identifier. It would be better to get the APP_NAME as an explicit parameter, but this can only be achieved by
            # passing it as a required environment variable in the Deployment Config (which is not desired).
            APP_NAME=$(echo $HOSTNAME | sed -r -e 's/(-[0-9]{1,10}){0,1}-[a-z0-9]{5}$//g')

            cat << EOF > $OUTPUT_FILE
# --- start/stanza STDOUT
[monitor://./logs/*.log]
disabled = false
followTail = 0
sourcetype = log4j
index = $SPLUNK_INDEX
_meta = environment::$POD_NAMESPACE application::${APP_NAME} nodetype::openshift
host = $HOSTNAME
# --- end/stanza

# --- start/stanza AUDIT
[monitor://./logs/*.audit.json]
disabled = false
followTail = 0
sourcetype = _json
index = $SPLUNK_AUDIT_INDEX
_meta = environment::$POD_NAMESPACE application::${APP_NAME} nodetype::openshift logtype::audit
host = $HOSTNAME
# --- end/stanza

# --- start/stanza ACCESS_LOG
[monitor://./logs/*.access]
disabled = false
followTail = 0
sourcetype = access_combined
index = $SPLUNK_INDEX
_meta = environment::$POD_NAMESPACE application::${APP_NAME} nodetype::openshift
host = $HOSTNAME
# --- end/stanza

# --- start/stanza GC LOG
[monitor://./logs/*.gc]
disabled = false
followTail = 0
sourcetype = gc_log
index = $SPLUNK_INDEX
_meta = environment::$POD_NAMESPACE application::${APP_NAME} nodetype::openshift
host = $HOSTNAME
# --- end/stanza
EOF
        else
            echo -e "${SPLUNK_STANZA}" | sed "s/host = openshift-host/host = $HOSTNAME/g;s/NAMESPACE-PLACEHOLDER/$POD_NAMESPACE/g;s/INDEX-PLACEHOLDER/$SPLUNK_INDEX/g" > $OUTPUT_FILE
        fi
    else
      echo -e "No SPLUNK_INDEX present, will not index in SPLUNK."
    fi
}


function load_aurora_config() {
  local SYMLINK_FOLDER=$1
  local CONFIG_BASE_DIR=$2
  local COMPLETE_VERSION=$3
  local APP_VERSION=$4


  local CONFIG_DIR=$CONFIG_BASE_DIR/configmap
  local SECRET_DIR=$CONFIG_BASE_DIR/secret

  CONFIG_VERSION=$(find_config_version ${COMPLETE_VERSION} ${APP_VERSION} ${CONFIG_DIR})
  if [ $? != 0 ]; then
      echo "No CONFIG mounted for this application"
  else
    echo "CONFIG configuration is pinned to $CONFIG_VERSION"

    ln -s $CONFIG_DIR/$CONFIG_VERSION.properties $SYMLINK_FOLDER/env.properties
    export AURORA_ENV_PREFIX=$CONFIG_DIR/$CONFIG_VERSION
    export AURORA_ENV_PROPERTIES=$SYMLINK_FOLDER/env.properties
    export_properties_file_as_env_variables "$AURORA_ENV_PROPERTIES"
    echo "The following env variables were set:"
    cat "$AURORA_ENV_PROPERTIES" | awk -F'=' '{ print $1 }'
  fi

  SECRET_VERSION=$(find_config_version {COMPLETE_VERSION} ${APP_VERSION} ${SECRET_DIR})
  if [ $? != 0 ]; then
    echo "No SECRET mounted for this application"
  else
    echo "SECRET configuration is pinned to $SECRET_VERSION"
    ln -s $SECRET_DIR/$SECRET_VERSION.properties $SYMLINK_FOLDER/secret.properties
    export AURORA_SECRET_PREFIX=$SECRET_DIR/$SECRET_VERSION
    export AURORA_SECRET=$SYMLINK_FOLDER/secret.properties

    export_properties_file_as_env_variables "$AURORA_SECRET"

    echo "The following env variables were set:"
    cat "$AURORA_SECRET" | awk -F'=' '{ print $1 }'
  fi

}

function export_properties_file_as_env_variables() {
  local PROPERTIES_FILE=$1
  local DONE=false
  until "$DONE" ;do
  read p || DONE=true
  if [[ "$p" == *"="* && "$p" != "#"* ]]
  then
    export "$p"
  fi
  done < "$PROPERTIES_FILE"
}

function find_config_version() {
 local COMPLETE_VERSION=$1
 local PATCH_VERSION=$2
 local CONFIG_LOCATION=$3

 MAJOR_VERSION=$(echo $PATCH_VERSION | awk -F. '{print $1}')
 MINOR_VERSION=$(echo $PATCH_VERSION | awk -F. '{print $1"."$2}')

 VERSIONS="$COMPLETE_VERSION $PATCH_VERSION $MINOR_VERSION $MAJOR_VERSION latest"

  for version in $(echo $VERSIONS); do
    if [ -f ${CONFIG_LOCATION}/$version.properties ]
    then
        echo $version
        return;
    fi
  done
  return -1
}

create_splunk_stanza "$SPLUNK_INDEX" $HOME/logs/application.splunk "$SPLUNK_STANZA"

if [ -z "$HTTP_PORT" ]; then
  export HTTP_PORT="8080"
fi

if [ -z "$DEPLOY_TYPE" ]; then
  export DEPLOY_TYPE="OpenShift"
fi

CONFIG_BASEDIR=$HOME/config
SYMLINK_DIR=$HOME

load_aurora_config $SYMLINK_DIR $CONFIG_BASEDIR $AURORA_VERSION $APP_VERSION