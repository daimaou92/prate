#/usr/bin/env bash
EX=`command -v protoc`
[ -z "$EX" ] && echo "protoc not in path or not installed" && exit 1
scriptDir() {
	P=`pwd`
	D="$(dirname $0)"
	if [[ $D == /* ]]; then
		echo $D
	elif [[ $D == \.* ]]; then
		J=`echo "$D" | sed 's/.//'`
		echo "${P}$J"
	else
		echo "${P}/$D"
	fi
}

S=`scriptDir`
P=`pwd`
cd $S

OUTDIR="../pb"
mkdir -p $OUTDIR
protoc -I="./" --go_out=$OUTDIR ./*.proto

cd $P
unset S
unset P
unset OUTDIR