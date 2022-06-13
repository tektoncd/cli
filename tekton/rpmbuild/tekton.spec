%define debug_package %{nil}
%define repo github.com/tektoncd/cli
%define version _VERSION_

Name:           tektoncd-cli
Version:        %{version}
Release:        1
Summary:        A command line interface for interacting with Tekton
License:        ASL 2.0
URL:            https://%{repo}
Source0:        https://%{repo}/archive/tkn_%{version}_Linux_x86_64.tar.gz
Source1:        manpages.tar.gz

%description
The Tekton Pipelines cli project provides a CLI for interacting with Tekton !

This rpm ships with the binary releases from github.com/tektoncd/cli/releases
and not compiling from the sources.

%prep
%setup -q -c
%setup -T -D -a 1

%install
install -D -m 0755 tkn %{buildroot}%{_bindir}/tkn

install -d %{buildroot}%{_datadir}/bash-completion/completions
./tkn completion bash > %{buildroot}%{_datadir}/bash-completion/completions/_tkn

install -d %{buildroot}%{_datadir}/zsh/site-functions
./tkn completion zsh > %{buildroot}%{_datadir}/zsh/site-functions/_tkn

mkdir -p %{buildroot}%{_mandir} && cp -a man1 %{buildroot}%{_mandir}

%files
%doc *.md
%license LICENSE
%{_bindir}/tkn
%{_datadir}/zsh/site-functions/*
%{_datadir}/bash-completion/completions/*
%{_mandir}/*

%changelog
* Mon Jun 13 2022 Chmouel Boudjnah <chmouel@chmouel.com> 0.24.0
- Don't compile, use binary from releases.

* Thu Sep 26 2019 Khurram Baig <kbaig@redhat.com> 0.4.0
- Updated Vendored Libraries

* Mon Sep 16 2019 Chmouel Boudjnah <chmouel@redhat.com> 0.3.1
- Install manpages
- Install shell completions
- Add docs
- Make version a macro.

* Tue Jul 02 2019 Khurram Baig <kbaig@redhat.com> 0.1.2
- Make Spec compliant to guidelines

* Thu Jun 20 2019 Khurram Baig <kbaig@redhat.com> 0.1.2
- Initial version of the rpm
