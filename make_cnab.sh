#!/bin/bash

git clone https://github.com/cisco-open/martian-bank-demo.git
cd martian-bank-demo
git apply ../martian-bank-demo.patch
cd ..
mkdir charts
mv martian-bank-demo/martianbank charts
rm -rf martian-bank-demo
porter build --verbosity=none --custom-dockerfile