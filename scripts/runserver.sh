#!/bin/bash

SSH_USER="SSH_USER"
SSH_PASSWORD="SSH_PASSWORD"
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
    send \"cd /home/$SSH_USER/mp_1_g35/membership-list && nohup go run main.go machine > output.log 2>&1 &\r\"

    expect \"~$\"
    send \"exit\r\"
    expect eof
    "
done
