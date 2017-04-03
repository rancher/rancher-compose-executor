from common_fixtures import *  # NOQA

from os import path
import os

import pytest
import cattle
import ConfigParser


@pytest.fixture(scope='session')
def client(admin_user_client, request):
    try:
        return cattle.from_env(url=os.environ['RANCHER_URL'],
                               access_key=os.environ['RANCHER_ACCESS_KEY'],
                               secret_key=os.environ['RANCHER_SECRET_KEY'])
    except KeyError:
        pass

    try:
        config = ConfigParser.ConfigParser()
        config.read(path.join(_base(), '../../tox.ini'))
        return cattle.from_env(url=config.get('rancher', 'url'),
                               access_key=config.get('rancher', 'access-key'),
                               secret_key=config.get('rancher', 'secret-key'))
    except ConfigParser.NoOptionError:
        pass

    return new_context(admin_user_client, request).client


def test_stack_create_upgrade_finish(client):
    name = 'project-' + random_str()
    rancher_compose = '''
.catalog:
  uuid: foo
'''
    template = '''
one:
  image: nginx

two:
  image: nginx
'''

    env = client.create_stack(name=name, dockerCompose=template,
                              rancherCompose=rancher_compose)
    env = client.wait_success(env)
    assert env.state == 'active'
    assert env.externalId is None

    names = set()
    for s in env.services():
        assert s.state == 'inactive'
        names.add(s.name)

    assert names == {'one', 'two'}

    env = client.wait_success(env.activateservices())
    for s in env.services():
        s = client.wait_success(s)
        assert s.state == 'active'

    rancher_compose = '''
.catalog:
  uuid: foo2
'''
    template = '''
one:
  image: nginx:2

two:
  image: nginx
'''

    # TODO: externalId should not be in upgrade
    env.upgrade(dockerCompose=template,
                rancherCompose=rancher_compose,
                externalId='foo2')
    env = client.wait_success(env, timeout=120)
    for s in env.services():
        s = client.wait_success(s)
        if s.name == 'one':
            assert s.state == 'upgraded'
        elif s.name == 'two':
            assert s.state == 'active'

    assert env.externalId == 'foo2'
    assert env.previousExternalId == ''

    env.finishupgrade()

    env = client.wait_success(env)
    for s in env.services():
        s = client.wait_success(s)
        assert s.state == 'active'

    assert env.externalId == 'foo2'
    assert env.previousExternalId is None


def test_stack_create_and_upgrade(client):
    name = 'project-' + random_str()
    rancher_compose = '''
.catalog:
  uuid: foo
'''
    template = '''
one:
  image: nginx

two:
  image: nginx
'''

    env = client.create_stack(name=name, dockerCompose=template,
                              environment={'a': 'b', 'd': 'e'},
                              rancherCompose=rancher_compose)
    env = client.wait_success(env)
    env = client.wait_success(env.activateservices())
    assert env.state == 'active'
    assert env.environment == {'a': 'b', 'd': 'e'}
    for s in env.services():
        s = client.wait_success(s)
        assert s.state == 'active'

    rancher_compose = '''
.catalog:
  uuid: foo2
'''
    template = '''
one:
  image: nginx:2

two:
  image: nginx
'''

    # TODO: externalId should not be in upgrade
    env = env.upgrade(dockerCompose=template,
                      environment={'a': 'x'},
                      rancherCompose=rancher_compose,
                      externalId='foo2')
    assert env.environment == {'a': 'b', 'd': 'e'}
    env = client.wait_success(env, timeout=120)
    assert env.state == 'upgraded'
    for s in env.services():
        s = client.wait_success(s)
        if s.name == 'one':
            assert s.state == 'upgraded'
    assert env.environment == {'a': 'x', 'd': 'e'}
    assert env.previousEnvironment == {'a': 'b', 'd': 'e'}

    env = env.rollback()
    env = client.wait_success(env, timeout=120)
    assert env.state == 'active'
    for s in env.services():
        s = client.wait_success(s)
        assert s.state == 'active'

    # TODO this should really be ''
    assert env.externalId == 'foo2'
    assert env.environment == {'a': 'b', 'd': 'e'}
    assert env.previousExternalId is None
    assert env.previousEnvironment is None


def test_stack_change_scale_upgrade(client):
    name = 'project-' + random_str()
    template = '''
one:
  image: nginx
'''
    rancher_compose = '''
one:
  scale: 2
'''
    env = client.create_stack(name=name, dockerCompose=template,
                              rancherCompose=rancher_compose)
    env = client.wait_success(env)
    env = client.wait_success(env.activateservices())
    assert env.state == 'active'
    s = find_one(env.services)
    assert s.launchConfig.imageUuid == 'docker:nginx'
    assert s.scale == 2

    template = '''
one:
  image: nginx:2
'''
    # Something else about the service needs to change too, like metadata
    # scale is ignore in the diff
    rancher_compose = '''
one:
  metadata:
    foo: bar
  scale: 4
'''
    env.upgrade(dockerCompose=template,
                rancherCompose=rancher_compose)
    env = client.wait_success(env, timeout=120)
    assert env.state == 'upgraded'
    s = find_one(env.services)
    assert s.launchConfig.imageUuid == 'docker:nginx:2'
    assert s.scale == 2


def test_stack_create_circles(client):
    name = 'project-' + random_str()
    template = '''
one:
  image: nginx
  links:
  - two

two:
  image: nginx
  links:
  - one
'''
    env = client.create_stack(name=name, dockerCompose=template)
    env = client.wait_success(env)

    for s in env.services():
        find_one(s.consumedservices)


def test_stack_variable_interpolation(client):
    name = 'project-' + random_str()
    rancher_compose = '''
.catalog:
  uuid: foo
  questions:
  - variable: d
    default: e
'''
    template = '''
one:
  image: nginx
  labels:
    a: $a
    d: $d
'''

    env = client.create_stack(name=name, dockerCompose=template,
                              environment={'a': 'b'},
                              rancherCompose=rancher_compose)
    env = client.wait_success(env)
    env = client.wait_success(env.activateservices())
    assert env.state == 'active'
    assert env.environment == {'a': 'b'}
    for s in env.services():
        s = client.wait_success(s)
        assert s.state == 'active'
        assert s.launchConfig.labels.a == 'b'
        assert s.launchConfig.labels.d == 'e'


def test_stack_templating(client):
    name = 'project-' + random_str()
    rancher_compose = '''
.catalog:
  uuid: foo
  questions:
  - variable: "image1"
    type: "string"
    default: "nginx"
  - variable: "image2"
    type: "string"
    default: "nginx"
  - variable: "b1"
    type: "boolean"
  - variable: "b2"
    type: "boolean"
'''
    template = '''
one:
  image: {{ .Values.image1 }}
  labels:
    {{- if eq .Values.b1 "true" }}
    label: "true"
    {{- else }}
    label: "false"
    {{- end }}
two:
  image: {{ .Values.image2 }}
  labels:
    {{- if eq .Values.b2 "true" }}
    label: "true"
    {{- else }}
    label: "false"
    {{- end }}
'''

    environment = {'image2': 'busybox', 'b1': 'true', 'b2': 'false'}
    env = client.create_stack(name=name, dockerCompose=template,
                              environment=environment,
                              rancherCompose=rancher_compose)
    env = client.wait_success(env)
    env = client.wait_success(env.activateservices())
    assert env.state == 'active'
    assert env.environment == environment
    for s in env.services():
        s = client.wait_success(s)
        assert s.state == 'active'
        if s.name == 'one':
            assert s.launchConfig.imageUuid == 'docker:nginx'
            assert s.launchConfig.labels.label == 'true'
        if s.name == 'two':
            assert s.launchConfig.imageUuid == 'docker:busybox'
            assert s.launchConfig.labels.label == 'false'


def test_release_variables(client):
    name = 'project-' + random_str()
    rancher_compose = '''
.catalog:
  version: 0.0.1
'''
    template = '''
one:
  image: nginx
  labels:
    a: '{{ .Release.Version }}'
    b: '{{ .Release.PreviousVersion }}'
'''

    env = client.create_stack(name=name, dockerCompose=template,
                              rancherCompose=rancher_compose)
    env = client.wait_success(env)
    env = client.wait_success(env.activateservices())
    assert env.state == 'active'
    for s in env.services():
        s = client.wait_success(s)
        assert s.state == 'active'

    rancher_compose = '''
.catalog:
  version: 0.0.2
'''

    # TODO: externalId should not be in upgrade
    env = env.upgrade(dockerCompose=template, rancherCompose=rancher_compose)
    env = client.wait_success(env, timeout=120)
    assert env.state == 'upgraded'
    for s in env.services():
        s = client.wait_success(s)
        if s.name == 'one':
            assert s.state == 'upgraded'
            assert s.launchConfig.labels['a'] == '0.0.2'
            assert s.launchConfig.labels['b'] == '0.0.1'


def test_storage_driver(client):
    template_legacy = '''
version: '2'
services:
  legacy-driver:
    image: nginx
    storage_driver:
      name: legacy-driver
      scope: environment
      volume_access_mode: multiHostRW
'''
    template = '''
version: '2'
storage_drivers:
  driver:
    image: nginx
    storage_driver:
      name: driver
      scope: environment
      volume_access_mode: multiHostRW
'''

    for template in (template_legacy, template):
        name = 'project-' + random_str()
        env = client.create_stack(name=name, dockerCompose=template)
        env = client.wait_success(env)
        env = client.wait_success(env.activateservices())
        assert env.state == 'active'

        services = env.services()
        assert len(services) == 1
        s = client.wait_success(services[0])
        assert s.state == 'active'
        assert s.type == 'storageDriverService'
        assert s.storageDriver.volumeAccessMode == 'multiHostRW'
        assert s.storageDriver.scope == 'environment'


def test_network_driver(client):
    template_legacy = '''
version: '2'
services:
  legacy-driver:
    image: nginx
    network_driver:
      name: legacy-driver
      default_network:
        name: vxlan
'''
    template = '''
version: '2'
network_drivers:
  driver:
    image: nginx
    network_driver:
      name: driver
      default_network:
        name: vxlan
'''

    for template in (template_legacy, template):
        name = 'project-' + random_str()
        env = client.create_stack(name=name, dockerCompose=template)
        env = client.wait_success(env)
        env = client.wait_success(env.activateservices())
        assert env.state == 'active'

        services = env.services()
        assert len(services) == 1
        s = client.wait_success(services[0])
        assert s.state == 'active'
        assert s.type == 'networkDriverService'
        assert s.networkDriver.defaultNetwork.name == 'vxlan'


def _base():
    return path.dirname(__file__)
