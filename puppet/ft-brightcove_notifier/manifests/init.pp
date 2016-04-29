class brightcove_notifier {

  $binary_name = "brightcove-notifier"
  $install_dir = "/usr/local/$binary_name"
  $binary_file = "$install_dir/$binary_name"
  $log_dir = "/var/log/apps"
  $brightcove_acc_id = hiera('brightcoveAccID')
  $brightcove_auth = hiera('brightcoveAuth')
  $cms_notifier_addr = hiera('cmsNotifierAddr')

  class { 'common_pp_up': }
  class { "${module_name}::monitoring": }
  class { "${module_name}::supervisord": }

  Class['common_pp_up'] -> Class["${module_name}::monitoring"] -> Class["${module_name}::supervisord"]

  user { $binary_name:
    ensure    => present,
  }

  file {
    $install_dir:
      mode    => "0664",
      ensure  => directory;

    $binary_file:
      ensure  => present,
      source  => "puppet:///modules/$module_name/$binary_name",
      mode    => "0755",
      require => File[$install_dir];

    $log_dir:
      ensure  => directory,
      mode    => "0664"
  }

  exec { 'restart_app':
    command     => "supervisorctl restart $binary_name",
    environment => [
      "BRIGHTCOVE_ACCOUNT_ID=$brightcove_acc_id",
      "BRIGHTCOVE_AUTH='$brightcove_auth'",
      "CMS_NOTIFIER=$cms_notifier_addr"
    ],
    path        => "/usr/bin:/usr/sbin:/bin",
    subscribe   => [
      File[$binary_file],
      Class["${module_name}::supervisord"]
    ],
    refreshonly => true
  }
}
