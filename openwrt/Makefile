#
# Copyright (C) 2024 OpenWrt.org
#
# This is free software, licensed under the GNU General Public License v2.
#

include $(TOPDIR)/rules.mk

PKG_NAME:=pansou-openwrt
PKG_VERSION:=1.0.0
PKG_RELEASE:=1

PKG_SOURCE_PROTO:=git
PKG_SOURCE_URL:=https://github.com/your-repo/pansou-openwrt.git
PKG_SOURCE_VERSION:=HEAD
PKG_MIRROR_HASH:=skip

PKG_LICENSE:=GPL-2.0
PKG_LICENSE_FILES:=LICENSE
PKG_MAINTAINER:=YourName <your@email.com>

PKG_BUILD_DEPENDS:=golang/host
PKG_BUILD_PARALLEL:=1
PKG_USE_MIPS16:=0

GO_PKG:=pansou-openwrt
GO_PKG_BUILD_PKG:=pansou-openwrt

include $(INCLUDE_DIR)/package.mk
include $(TOPDIR)/feeds/packages/lang/golang/golang-package.mk

define Package/pansou-openwrt
  SECTION:=net
  CATEGORY:=Network
  TITLE:=Network disk search engine for OpenWrt
  URL:=https://github.com/your-repo/pansou-openwrt
  DEPENDS:=$(GO_ARCH_DEPENDS) +ca-bundle
endef

define Package/pansou-openwrt/description
  PanSou is a network disk search engine that supports multiple cloud storage 
  platforms including Baidu Netdisk, Aliyun Drive, Quark Drive, and Telegram channels.
  
  Features:
  - Multiple search plugins support
  - Telegram channel search (with network detection)
  - Support various cloud storage types
  - LuCI web interface
  - Configurable search parameters
endef

define Package/pansou-openwrt/conffiles
/etc/config/pansou
/etc/pansou/config.yaml
endef

define Build/Prepare
	$(call Build/Prepare/Default)
endef

define Build/Compile
	$(call GoPackage/Build/Compile)
endef

define Package/pansou-openwrt/install
	$(INSTALL_DIR) $(1)/usr/bin
	$(INSTALL_BIN) $(GO_PKG_BUILD_BIN_DIR)/pansou-openwrt $(1)/usr/bin/

	$(INSTALL_DIR) $(1)/etc/init.d
	$(INSTALL_BIN) ./files/pansou.init $(1)/etc/init.d/pansou

	$(INSTALL_DIR) $(1)/etc/config
	$(INSTALL_CONF) ./files/pansou.config $(1)/etc/config/pansou

	$(INSTALL_DIR) $(1)/etc/pansou
endef

$(eval $(call GoBinPackage,pansou-openwrt))
$(eval $(call BuildPackage,pansou-openwrt))
