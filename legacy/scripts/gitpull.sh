#!/bin/bash

# Replace with your SSH username, SSH password, Git username, and Git password
SSH_USER="SSH_USERNAME"
SSH_PASSWORD="SSH_PASSWORD"
GIT_USERNAME="GIT_USERNAME"
GIT_PASSWORD="GIT_PASSWORD"

VMS=(
  "fa24-cs425-3501.cs.illinois.edu"
  "fa24-cs425-3502.cs.illinois.edu"
  "fa24-cs425-3503.cs.illinois.edu"
  "fa24-cs425-3504.cs.illinois.edu"
  "fa24-cs425-3505.cs.illinois.edu"
  "fa24-cs425-3506.cs.illinois.edu"
  "fa24-cs425-3507.cs.illinois.edu"
  "fa24-cs425-3508.cs.illinois.edu"
  "fa24-cs425-3509.cs.illinois.edu"
  "fa24-cs425-3510.cs.illinois.edu"
)

for VM in "${VMS[@]}"
do
    echo "Connecting to $VM"

    expect -c "
    spawn sshpass -p \"$SSH_PASSWORD\" ssh -o StrictHostKeyChecking=no $SSH_USER@$VM

    expect \"~$\"
    send \"cd /home/$SSH_USER/mp_1_g35 && git pull\r\"

    expect \"Username for 'https://github.com':\"
    send \"$GIT_USERNAME\r\"

    expect \"Password for 'https://$GIT_USERNAME@github.com':\"
    send \"$GIT_PASSWORD\r\"

    expect \"~$\"
    send \"exit\r\"
    expect eof
    "
done
