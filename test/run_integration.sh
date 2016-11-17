#!/bin/bash
source /home/vagrant/.profile
rm -rf /home/vagrant/src/acme-dns/*
cp -R /vagrant/* /home/vagrant/src/acme-dns/
cd /home/vagrant/src/acme-dns/
go get
go test -postgres
