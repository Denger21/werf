name 'dappdeps-base'

license 'MIT'
license_file 'https://github.com/flant/dappdeps-base/blob/master/LICENSE.txt'

dependency 'bash'

build do
  link "#{install_dir}/embedded/bin", "#{install_dir}/bin"
end
