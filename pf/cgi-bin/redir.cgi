#!/usr/bin/perl
#use Data::Dumper;
use strict;
use warnings;

use CGI;
use CGI::Carp qw( fatalsToBrowser );
use CGI::Session;
use Log::Log4perl;

use lib '/usr/local/pf/lib';
use pf::config;
use pf::iplog;
use pf::util;
use pf::web;
# not SUID now!
#use pf::rawip;
use pf::node;
use pf::class;
use pf::violation;

Log::Log4perl->init('/usr/local/pf/conf/log.conf');
my $logger = Log::Log4perl->get_logger('redir.cgi');
Log::Log4perl::MDC->put('proc', 'redir.cgi');
Log::Log4perl::MDC->put('tid', 0);

my $cgi = new CGI;
my $session = new CGI::Session(undef, $cgi, {Directory=>'/tmp'});

my $result;
my $ip              = $cgi->remote_addr();
my $destination_url = $cgi->param("destination_url");
my $enable_menu     = $cgi->param("enable_menu");
my $mac             = ip2mac($ip);
my %tags;

# valid mac?
if (!valid_mac($mac)) {
  $logger->info("$ip not resolvable, generating error page");
  generate_error_page($cgi, $session, "error: not found in the database");
  exit(0);
}
$logger->info("$mac being redirected");

# registration auth request?
if (defined($cgi->param('mode')) && $cgi->param('auth')) {
 my $type=$cgi->param('auth');
 if ($type eq "skip"){
    $logger->info("User is trying to skip redirecting to release.cgi");
    print $cgi->redirect("/cgi-bin/release.cgi?mode=skip&destination_url=$destination_url");    
  }else{
    $logger->info("redirecting to register-$type.cgi for reg authentication");
    print $cgi->redirect("/cgi-bin/register-$type.cgi?mode=register&destination_url=$destination_url");
  }
}


#check to see if node needs to be registered
#
my $unreg = node_unregistered($mac);
if ($unreg && isenabled($Config{'trapping'}{'registration'})){
  $logger->info("$mac redirected to registration page");
  generate_registration_page($cgi, $session, $destination_url,$mac);
  exit(0);
} 

# check violation 
#
my $violation = violation_view_top($mac);
if ($violation){
  if ($unreg && $Config{'trapping'}{'registration'} =~ /^onviolation$/) {
    $logger->info("$mac redirected to registration page");
    generate_registration_page($cgi, $session, $destination_url,$mac);
    exit(0);
  }
  my $vid=$violation->{'vid'};
  my $class=class_view($vid);
  # enable button
  if ($enable_menu) {
    $logger->info("enter enable_menu");
    generate_enabler_page($cgi, $session, $destination_url, $vid, $class->{'button_text'});
  } elsif  ($class->{'auto_enable'} eq 'Y'){
    $logger->info("auth_enable =  Y");
    generate_redirect_page($cgi, $session, $class->{'url'}, $destination_url);
  } else {
    $logger->info("no button");
    # no enable button 
    print $cgi->redirect($class->{'url'});
  }
} else {
  $logger->info("$mac already registered or registration disabled, freeing mac");
  if ($Config{'network'}{'mode'} =~ /passive/i) {
    my $cmd = $install_dir."/bin/pfcmd manage freemac $mac";
    my $output = qx/$cmd/;
  }
  $logger->info("freed $mac and redirecting to ".$Config{'trapping'}{'redirecturl'});
  print $cgi->redirect($Config{'trapping'}{'redirecturl'});
}
