# -*- mode: ruby -*-
# vi: set ft=ruby :

# Vagrantfile for running integration tests with PostgreSQL

VAGRANTFILE_API_VERSION = "2"

$ubuntu_setup_script = <<SETUP_SCRIPT
apt-get update
apt-get install -y vim build-essential postgresql postgresql-contrib git
echo "Downloading and installing Go 1.7.3"
curl -s -o /tmp/go.tar.gz https://storage.googleapis.com/golang/go1.7.3.linux-amd64.tar.gz
tar -C /usr/local -xzf /tmp/go.tar.gz
echo "export PATH=$PATH:/usr/local/go/bin" >> /home/vagrant/.profile
echo "export GOPATH=/home/vagrant" >> /home/vagrant/.profile
mkdir -p /home/vagrant/src/acme-dns
chown -R vagrant /home/vagrant/src
cp /vagrant/test/run_integration.sh /home/vagrant
bash /vagrant/test/pgsql.sh
echo "\n-------------------------------------------------------------"
echo "To run integration tests run, /home/vagrant/run_integration.sh"
SETUP_SCRIPT

Vagrant.configure(VAGRANTFILE_API_VERSION) do |config|

  config.vm.define "ad-ubuntu-trusty", primary: true do |ad_ubuntu_trusty|
    ad_ubuntu_trusty.vm.box = "ubuntu/trusty64"
    ad_ubuntu_trusty.vm.provision "shell", inline: $ubuntu_setup_script
    ad_ubuntu_trusty.vm.network "forwarded_port", guest: 8080, host: 8008
    ad_ubuntu_trusty
    ad_ubuntu_trusty.vm.provider "virtualbox" do |v|
      v.memory = 2048
    end
  end

end
