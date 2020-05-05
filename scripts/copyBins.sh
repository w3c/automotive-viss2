#!/bin/bash  

helpFunction()
{
   echo ""
   echo "Usage: $0 -u user -s server -p path"
   echo -e "\t-u Server username"
   echo -e "\t-s Server URL"
   echo -e "\t-p folder/path on the server (/home/user/folder/)"
   exit 1 # Exit script after printing help
}

while getopts "u:s:p:" opt
do
   case "$opt" in
      u ) username="$OPTARG" ;;
      s ) server="$OPTARG" ;;
      p ) path="$OPTARG" ;;
      ? ) helpFunction ;; # Print helpFunction in case parameter is non-existent
   esac
done

# Print helpFunction in case parameters are empty
if [ -z "$username" ] || [ -z "$server" ] || [ -z "$path" ]
then
   echo "Some or all of the parameters are empty";
   helpFunction
fi

cmd="rsync -avzhe ssh" 
echo $(pwd)
$cmd puppi.sh $username@$server:$path
cd .. #move to project root 
dir=$(pwd)
echo $dir
cd server/server-core  && go build -o servercore
$cmd servercore vss_rel_2.0.0-alpha+006.cnative $username@$server:$path
cd $dir

cd server/servicemgr && go build -o servicemgr
$cmd servicemgr $username@$server:$path
cd $dir

cd server/wsmgr && go build -o wsmgr
$cmd wsmgr $username@$server:$path
cd $dir

cd server/httpmgr && go build -o httpmgr
$cmd httpmgr $username@$server:$path
cd $dir

cd client/client-1.0/Go && go build -o agtserver
$cmd agtserver $username@$server:$path
cd $dir

cd server/atserver && go build -o atserver
$cmd atserver $username@$server:$path
cd $dir

