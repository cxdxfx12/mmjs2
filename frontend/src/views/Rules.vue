<template>
  <div class="rules-page">
    <!-- 顶部：全局规则面板（可折叠） -->
    <div class="global-panel" :class="{ collapsed: globalCollapsed }">
      <div class="global-header" @click="globalCollapsed = !globalCollapsed">
        <div class="global-title">
          <el-icon :size="18"><Setting /></el-icon>
          <span>全局规则设置</span>
        </div>
        <el-icon :size="16" class="collapse-icon" :class="{ rotated: !globalCollapsed }"><ArrowDown /></el-icon>
      </div>
      <div class="global-body" v-show="!globalCollapsed">
        <!-- 第一行：保底规则 -->
        <div class="global-row">
          <div class="global-row-label"><el-tag type="warning" size="small" effect="plain">保底规则</el-tag></div>
          <div class="global-fields">
            <label>默认首重<em>(kg)</em></label>
            <el-input-number v-model="grForm.default_first_weight" :min="0.1" :step="0.5" :precision="1" size="small" controls-position="right" />
            <label>默认首重单价<em>(元)</em></label>
            <el-input-number v-model="grForm.default_first_price" :min="0" :step="0.1" :precision="2" size="small" controls-position="right" />
            <label>默认续重单价<em>(元)</em></label>
            <el-input-number v-model="grForm.default_cont_price" :min="0" :step="0.05" :precision="2" size="small" controls-position="right" />
            <label>默认保底价<em>(元)</em></label>
            <el-input-number v-model="grForm.default_min_fee" :min="0" :step="0.1" :precision="2" size="small" controls-position="right" />
            <label>无重量默认价<em>(元)</em></label>
            <el-input-number v-model="grForm.no_weight_price" :min="0" :step="0.1" :precision="2" size="small" controls-position="right" />
          </div>
        </div>
        <!-- 第二行：加价规则 -->
        <div class="global-row">
          <div class="global-row-label"><el-tag type="danger" size="small" effect="plain">加价规则</el-tag></div>
          <div class="global-fields">
            <label>固定加价<em>(元/单)</em></label>
            <el-input-number v-model="grForm.markup_fixed" :min="0" :step="0.1" :precision="2" size="small" controls-position="right" />
            <label>百分比加价<em>(%)</em></label>
            <el-input-number v-model="grForm.markup_percent" :min="0" :step="0.5" :precision="1" size="small" controls-position="right" />
            <span class="gr-desc">对每单运费额外向上加收，在所有规则计算完毕后生效</span>
          </div>
        </div>
        <!-- 第三行：省份加价 -->
        <div class="global-row province-surcharge-row">
          <div class="global-row-label"><el-tag type="warning" size="small" effect="plain">省份加价</el-tag></div>
          <div class="province-surcharge-area">
            <div class="ps-list" v-if="provinceSurcharges.length > 0">
              <el-tag
                v-for="ps in provinceSurcharges"
                :key="ps.id"
                closable
                size="small"
                type="warning"
                effect="plain"
                class="ps-tag"
                @close="removeProvinceSurcharge(ps.id)"
              >
                {{ ps.province_name }} +¥{{ ps.surcharge }}
              </el-tag>
            </div>
            <div class="ps-add">
              <el-select v-model="psForm.province_name" filterable placeholder="选择省份" size="small" style="width:120px">
                <el-option v-for="p in PROVINCES" :key="p" :label="p" :value="p" />
              </el-select>
              <el-input-number v-model="psForm.surcharge" :min="0" :step="0.5" :precision="2" size="small" controls-position="right" style="width:110px" />
              <span class="ps-unit">元/票</span>
              <el-button type="warning" size="small" plain @click="addProvinceSurcharge" :disabled="!psForm.province_name">添加</el-button>
            </div>
          </div>
        </div>
        <div class="global-actions">
          <el-button type="primary" size="small" @click="saveGlobal" :loading="savingGlobal">保存全局设置</el-button>
        </div>
      </div>
    </div>

    <!-- 主体：左右分栏 -->
    <div class="rules-main">
      <!-- 左侧：客户列表 -->
      <div class="left-panel">
        <div class="left-header">
          <h4><el-icon><UserFilled /></el-icon> 客户列表</h4>
          <el-button type="primary" size="small" @click="showAddCustomer = true">
            <el-icon><Plus /></el-icon>新增
          </el-button>
        </div>
        <div class="left-search">
          <el-input v-model="customerSearch" placeholder="搜索客户..." size="small" clearable :prefix-icon="Search" />
        </div>
        <div class="left-actions">
          <el-upload :show-file-list="false" :auto-upload="false" accept=".xlsx,.xls" :on-change="handleImport" :disabled="importing">
            <el-button size="small" :loading="importing">
              <el-icon><Upload /></el-icon>导入
            </el-button>
          </el-upload>
          <el-button size="small" @click="handleExport" :loading="exporting">
            <el-icon><Download /></el-icon>导出
          </el-button>
          <el-button size="small" @click="store.downloadTemplate()">
            <el-icon><Document /></el-icon>模板
          </el-button>
        </div>
        <div class="customer-list" v-loading="loadingCustomers" @dragover.prevent>
          <div
            v-for="c in filteredCustomers"
            :key="c.name"
            class="customer-item"
            :class="{ active: activeCustomer === c.name, 'drop-over': dragOverTarget === c.name }"
            draggable="true"
            @click="selectCustomer(c.name)"
            @dragstart="onDragStart($event, c.name)"
            @dragend="onDragEnd"
            @dragleave="onDragLeave(c.name)"
          >
            <div class="ci-left">
              <el-icon :size="18"><Avatar /></el-icon>
              <span class="ci-name">{{ c.name }}</span>
            </div>
            <div class="ci-right">
              <el-tag size="small" effect="plain" round>{{ c.rule_count }}条</el-tag>
              <el-popconfirm title="将删除该客户所有规则，确定？" @confirm="handleDeleteCustomer(c.name)">
                <template #reference>
                  <el-icon class="ci-del" :size="14" @click.stop><Close /></el-icon>
                </template>
              </el-popconfirm>
            </div>
          </div>
          <div v-if="filteredCustomers.length === 0" class="empty-customers">
            暂无客户，请新增或导入
          </div>
        </div>
      </div>

      <!-- 右侧：规则编辑区 -->
      <div class="right-panel">
        <template v-if="activeCustomer">
          <div class="right-header">
            <div class="rh-left">
              <span class="rh-customer">{{ activeCustomer }}</span>
              <el-tag v-if="customerRules.length > 0" size="small" effect="dark" round>{{ customerRules.length }}条规则</el-tag>
              <el-radio-group v-model="viewMode" size="small" style="margin-left: 12px">
                <el-radio-button value="zone">阶段报价</el-radio-button>
                <el-radio-button value="list">列表视图</el-radio-button>
              </el-radio-group>
            </div>
            <div class="rh-actions">
              <el-button type="success" size="small" plain @click="openZoneTemplateDlg">
                <el-icon><Grid /></el-icon>区域模板生成
              </el-button>
              <el-button type="danger" size="small" plain :disabled="selectedRuleIds.length===0" @click="batchDelete">批量删除</el-button>
              <el-button type="primary" size="small" @click="openRuleDlg(null)">
                <el-icon><Plus /></el-icon>新增规则
              </el-button>
            </div>
          </div>

          <!-- 阶段报价视图 -->
          <div v-if="viewMode === 'zone'" class="zone-view-wrap" v-loading="loadingRules">
            <template v-if="customerRules.length > 0">
              <div class="zone-table-container">
                <el-table :data="bracketRules" stripe border size="small" max-height="calc(100vh - 460px)" class="zone-big-table">
                  <el-table-column prop="province" label="省份" width="80" fixed="left">
                    <template #default="{row}">
                      <span :class="{ 'rule-disabled': row.is_enabled !== 1 }">{{ row.province || '全国' }}</span>
                    </template>
                  </el-table-column>
                  <el-table-column label="区域" width="72" align="center">
                    <template #default="{row}">
                      <el-tag :type="zoneTagType(extractZoneOrder(row.zone_name))" effect="plain" size="small">{{ row.zone_name || '—' }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="定价模式" width="90" align="center">
                    <template #default="{row}">
                      <el-tag :type="row.calc_mode==='bracket'?'warning':'info'" effect="plain" size="small">
                        {{ row.calc_mode==='bracket'?'阶梯':'标准' }}
                      </el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="0-0.5kg" width="84" align="center">
                    <template #default="{row}"><span class="price-fix">{{ getBracketFixedPrice(row, 0, 0.5) }}</span></template>
                  </el-table-column>
                  <el-table-column label="0.5-1kg" width="84" align="center">
                    <template #default="{row}"><span class="price-fix">{{ getBracketFixedPrice(row, 0.5, 1) }}</span></template>
                  </el-table-column>
                  <el-table-column label="1-2kg" width="80" align="center">
                    <template #default="{row}"><span class="price-fix">{{ getBracketFixedPrice(row, 1, 2) }}</span></template>
                  </el-table-column>
                  <el-table-column label="2-3kg" width="80" align="center">
                    <template #default="{row}"><span class="price-fix">{{ getBracketFixedPrice(row, 2, 3) }}</span></template>
                  </el-table-column>
                  <el-table-column label="首重(3-30)" width="90" align="center">
                    <template #default="{row}"><span class="price-first">{{ getBracketFirstPrice(row, 3, 30) }}</span></template>
                  </el-table-column>
                  <el-table-column label="续重(3-30)" width="90" align="center">
                    <template #default="{row}"><span class="price-cont">{{ getBracketContPrice(row, 3, 30) }}</span></template>
                  </el-table-column>
                  <el-table-column label="首重(30+)" width="90" align="center">
                    <template #default="{row}"><span class="price-first">{{ getBracketFirstPrice(row, 30, 0) }}</span></template>
                  </el-table-column>
                  <el-table-column label="续重(30+)" width="90" align="center">
                    <template #default="{row}"><span class="price-cont">{{ getBracketContPrice(row, 30, 0) }}</span></template>
                  </el-table-column>
                  <el-table-column label="保底费" width="80" align="center">
                    <template #default="{row}">{{ row.min_fee > 0 ? '¥' + row.min_fee : '—' }}</template>
                  </el-table-column>
                  <el-table-column label="续重单位" width="90" align="center">
                    <template #default="{row}">
                      <el-tag size="small" effect="plain" type="info">{{ getContModeLabel(row.cont_mode) }}</el-tag>
                    </template>
                  </el-table-column>
                  <el-table-column label="拉均重/偏差加价" width="240" align="center">
                    <template #default>
                      <div class="avgweight-cell" v-if="avgWeightRule">
                        <div class="aw-top">
                          <span class="aw-base">基准 {{ avgWeightRule.base_weight }}kg</span>
                          <el-switch :model-value="avgWeightRule.is_enabled===1" size="small" @change="toggleAvgWeightRule" />
                        </div>
                        <div class="aw-bottom">
                          步长 <span class="aw-step">{{ avgWeightRule.step_weight }}kg</span>
                          <span class="aw-sep">/</span>
                          <span class="aw-price">+¥{{ avgWeightRule.step_price }}</span>
                        </div>
                        <div class="aw-edit">
                          <el-button link type="primary" size="small" @click="openAvgWeightDlg">
                            <el-icon><Setting /></el-icon>&nbsp;配置
                          </el-button>
                          <span class="aw-scope-tag" v-if="avgWeightRule.scope_type==='global'">全局</span>
                          <span class="aw-scope-tag aw-customer" v-else>客户专属</span>
                        </div>
                      </div>
                      <div v-else class="aw-empty-wrap">
                        <el-button type="primary" size="small" plain @click="openAvgWeightDlg">
                          <el-icon><Plus /></el-icon>&nbsp;设置拉均重
                        </el-button>
                      </div>
                    </template>
                  </el-table-column>
                  <el-table-column label="启用" width="60" align="center" fixed="right">
                    <template #default="{row}"><el-switch :model-value="row.is_enabled===1" size="small" @change="toggleRule(row)"/></template>
                  </el-table-column>
                  <el-table-column label="操作" width="110" align="center" fixed="right">
                    <template #default="{row}">
                      <el-button link type="primary" size="small" @click="openRuleDlg(row)">编辑</el-button>
                      <el-popconfirm title="确定删除？" @confirm="handleDeleteRule(row.id)">
                        <template #reference><el-button link type="danger" size="small">删除</el-button></template>
                      </el-popconfirm>
                    </template>
                  </el-table-column>
                </el-table>
              </div>
            </template>
            <div v-else class="empty-zone">
              <el-empty description="暂无规则，点击「区域模板生成」快速创建" :image-size="80" />
            </div>
          </div>

          <!-- 列表视图 -->
          <div v-else class="rule-table-wrap">
            <el-table :data="customerRules" stripe border size="small" v-loading="loadingRules" max-height="calc(100vh - 420px)">
              <el-table-column type="selection" width="36" />
              <el-table-column prop="province" label="省份" width="100">
                <template #default="{row}"><span :class="{ 'prov-all': !row.province }">{{ row.province || '全国' }}</span></template>
              </el-table-column>
              <el-table-column label="区域" width="70" align="center">
                <template #default="{row}">
                  <el-tag :type="zoneTagType(extractZoneOrder(row.zone_name))" effect="plain" size="small">{{ row.zone_name || '—' }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="定价模式" width="80" align="center">
                <template #default="{row}">
                  <el-tag :type="row.calc_mode==='bracket'?'warning':'info'" effect="plain" size="small">
                    {{ row.calc_mode==='bracket'?'阶梯':'标准' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="续重模式" width="100">
                <template #default="{row}">
                  <el-tag :type="row.cont_mode==='hundred_gram'?'warning':'info'" size="small" effect="plain">
                    {{ row.cont_mode==='actual_weight'?'实际重量':(row.cont_mode==='hundred_gram'?'百克续重':'整kg续重') }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="首重(kg)" width="80"><template #default="{row}">{{ row.first_weight || (row.calc_mode==='bracket' && row.brackets && row.brackets.length>0 ? (row.brackets.find(b=>b.calc_type==='first_cont')?.first_weight || 1) : 1) }}</template></el-table-column>
              <el-table-column label="首重单价" width="90">
                <template #default="{row}">
                  <span v-if="row.calc_mode==='bracket' && row.brackets && row.brackets.length>0">
                    {{ getBracketFirstPrice(row, 3, 30) }}
                  </span>
                  <span v-else>¥{{ row.first_price }}</span>
                </template>
              </el-table-column>
              <el-table-column label="续重单价" width="120">
                <template #default="{row}">
                  <span v-if="row.calc_mode==='bracket' && row.brackets && row.brackets.length>0">
                    {{ getBracketContPrice(row, 3, 30) }}<span class="unit">/{{ row.cont_mode==='hundred_gram'?'百克':'kg' }}</span>
                  </span>
                  <span v-else>¥{{ row.cont_price }}<span class="unit">/{{ row.cont_mode==='hundred_gram'?'百克':'kg' }}</span></span>
                </template>
              </el-table-column>
              <el-table-column label="区间价格" width="160" show-overflow-tooltip>
                <template #default="{row}">
                  <span v-if="row.calc_mode==='bracket' && row.brackets && row.brackets.length>0" class="bracket-prices">
                    {{ row.brackets.filter(b=>b.calc_type==='fixed').map(b=>`${b.weight_from}-${b.weight_to}kg:¥${b.fixed_price}`).join('; ') }}
                  </span>
                  <span v-else class="empty-text">—</span>
                </template>
              </el-table-column>
              <el-table-column label="保底价" width="80"><template #default="{row}">{{ row.min_fee > 0 ? '¥' + row.min_fee : '—' }}</template></el-table-column>
              <el-table-column label="最高价" width="80"><template #default="{row}">{{ row.max_fee > 0 ? '¥' + row.max_fee : '—' }}</template></el-table-column>
              <el-table-column label="附加费" width="80"><template #default="{row}">{{ row.surcharge > 0 ? '¥' + row.surcharge : '—' }}</template></el-table-column>
              <el-table-column label="规则类型" width="85">
                <template #default="{row}">
                  <el-tag :type="row.rule_type==='campaign'?'danger':'success'" size="small" effect="plain">
                    {{ row.rule_type==='campaign'?'活动':'客户' }}
                  </el-tag>
                </template>
              </el-table-column>
              <el-table-column label="启用" width="60"><template #default="{row}"><el-switch :model-value="row.is_enabled===1" size="small" @change="toggleRule(row)"/></template></el-table-column>
              <el-table-column label="备注" min-width="100" show-overflow-tooltip><template #default="{row}">{{ row.remark }}</template></el-table-column>
              <el-table-column label="操作" width="120" fixed="right">
                <template #default="{row}">
                  <el-button link type="primary" size="small" @click="openRuleDlg(row)">编辑</el-button>
                  <el-popconfirm title="确定删除？" @confirm="handleDeleteRule(row.id)">
                    <template #reference><el-button link type="danger" size="small">删除</el-button></template>
                  </el-popconfirm>
                </template>
              </el-table-column>
            </el-table>
          </div>
        </template>
        <!-- 未选客户时 -->
        <div v-else class="no-customer">
          <el-icon :size="48" color="#c0c4cc"><UserFilled /></el-icon>
          <p>选择左侧客户查看其计费规则</p>
          <p class="sub">或点击「新增」创建新客户</p>
        </div>
      </div>
    </div>

    <!-- 新增客户弹窗 -->
    <el-dialog v-model="showAddCustomer" title="新增客户" width="400px" destroy-on-close @closed="newCustomerName=''">
      <el-form label-width="80px">
        <el-form-item label="客户名称">
          <el-input v-model="newCustomerName" placeholder="请输入客户名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="showAddCustomer=false">取消</el-button>
        <el-button type="primary" @click="confirmAddCustomer" :disabled="!newCustomerName.trim()">确认</el-button>
      </template>
    </el-dialog>

    <!-- 规则编辑弹窗 -->
    <el-dialog v-model="ruleDlgVisible" :title="editingRule?.id?'编辑规则':'新增规则'" width="700px" destroy-on-close>
      <el-form :model="ruleForm" label-width="100px" size="default">
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="规则类型">
              <el-select v-model="ruleForm.rule_type" style="width:100%">
                <el-option label="客户规则" value="customer"/>
                <el-option label="活动规则" value="campaign"/>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="省份">
              <el-select v-model="ruleForm.province" filterable clearable placeholder="空=全国" style="width:100%">
                <el-option v-for="p in PROVINCES" :key="p" :label="p" :value="p"/>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="12">
            <el-form-item label="定价模式">
              <el-select v-model="ruleForm.calc_mode" style="width:100%">
                <el-option label="阶梯" value="bracket"/>
                <el-option label="标准" value="simple"/>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="续重模式">
              <el-select v-model="ruleForm.cont_mode" style="width:100%">
                <el-option label="实际重量" value="actual_weight"/>
                <el-option label="整kg续重" value="full_kg"/>
                <el-option label="百克续重" value="hundred_gram"/>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="首重单价(元)">
              <el-input-number v-model="ruleForm.first_price" :min="0" :step="0.1" :precision="2" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="续重单价(元)">
              <el-input-number v-model="ruleForm.cont_price" :min="0" :step="0.05" :precision="2" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="偏远附加费(元)">
              <el-input-number v-model="ruleForm.surcharge" :min="0" :precision="2" style="width:100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="16">
          <el-col :span="8">
            <el-form-item label="保底价(元)">
              <el-input-number v-model="ruleForm.min_fee" :min="0" :precision="2" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="最高价(元)">
              <el-input-number v-model="ruleForm.max_fee" :min="0" :precision="2" style="width:100%" />
            </el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="启用">
              <el-switch v-model="ruleForm.is_enabled" :active-value="1" :inactive-value="0" size="default" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="备注">
          <el-input v-model="ruleForm.remark" placeholder="备注信息" />
        </el-form-item>
        <!-- 活动规则特有字段 -->
        <template v-if="ruleForm.rule_type==='campaign'">
          <el-row :gutter="16">
            <el-col :span="12">
              <el-form-item label="活动名称"><el-input v-model="ruleForm.campaign_name" /></el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="开始日期"><el-input type="date" v-model="ruleForm.campaign_start" /></el-form-item>
            </el-col>
            <el-col :span="6">
              <el-form-item label="结束日期"><el-input type="date" v-model="ruleForm.campaign_end" /></el-form-item>
            </el-col>
          </el-row>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="ruleDlgVisible=false">取消</el-button>
        <el-button type="primary" @click="saveRule" :loading="savingRule">保存</el-button>
      </template>
    </el-dialog>

    <!-- 区域模板生成弹窗 -->
    <el-dialog v-model="zoneTemplateDlgVisible" title="按区域模板生成规则" width="900px" destroy-on-close class="zone-template-dlg">
      <div class="zt-info">
        <el-alert type="info" :closable="false" show-icon size="small">
          <template #title>
            将为客户「<b>{{ activeCustomer }}</b>」按 6 区体系生成区间计费规则（含港澳台六区）。生成前会自动删除该客户已有的区域型/区间型规则。
          </template>
        </el-alert>
      </div>
      <div class="zt-toolbar">
        <div class="zt-left">
          <span class="zt-label">定价模式：</span>
          <el-select v-model="zoneForm.calc_mode" size="default" style="width:120px">
            <el-option label="阶梯" value="bracket" />
            <el-option label="标准" value="simple" />
          </el-select>
          <span class="zt-label ml-2">续重模式：</span>
          <el-select v-model="zoneForm.cont_mode" size="default" style="width:140px">
            <el-option label="实际重量" value="actual_weight" />
            <el-option label="整kg续重" value="full_kg" />
            <el-option label="百克续重" value="hundred_gram" />
          </el-select>
        </div>
        <div class="zt-right">
          <el-button size="default" @click="loadSamplePrice">加载参考价</el-button>
        </div>
      </div>
      <div class="zt-table-wrap">
        <el-table :data="zonePriceList" border size="default" class="zt-table">
          <el-table-column label="区域" width="90" align="center">
            <template #default="{row}">
              <el-tag :type="zoneTagType(row.zone_order)" effect="plain" size="small">{{ row.zone_name }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="包含省份" min-width="140">
            <template #default="{row}">
              <span class="zt-provinces">{{ (row.provinces || []).join('、') }}</span>
            </template>
          </el-table-column>
          <el-table-column label="0~0.5kg" width="95" align="center">
            <template #default="{row}">
              <el-input-number v-model="row.price_0_05" :min="0" :step="0.1" :precision="2" size="small" controls-position="right" style="width:90px" />
            </template>
          </el-table-column>
          <el-table-column label="0.5~1kg" width="95" align="center">
            <template #default="{row}">
              <el-input-number v-model="row.price_05_1" :min="0" :step="0.1" :precision="2" size="small" controls-position="right" style="width:90px" />
            </template>
          </el-table-column>
          <el-table-column label="1~2kg" width="95" align="center">
            <template #default="{row}">
              <el-input-number v-model="row.price_1_2" :min="0" :step="0.1" :precision="2" size="small" controls-position="right" style="width:90px" />
            </template>
          </el-table-column>
          <el-table-column label="2~3kg" width="95" align="center">
            <template #default="{row}">
              <el-input-number v-model="row.price_2_3" :min="0" :step="0.1" :precision="2" size="small" controls-position="right" style="width:90px" />
            </template>
          </el-table-column>
          <el-table-column label="3~30kg (首重+续重)" width="180" align="center">
            <template #default="{row}">
              <div class="zt-range">
                <el-input-number v-model="row.first_3_30" :min="0" :step="0.1" :precision="2" size="small" controls-position="right" style="width:78px" placeholder="首重" />
                <span class="zt-plus">+</span>
                <el-input-number v-model="row.cont_3_30" :min="0" :step="0.05" :precision="2" size="small" controls-position="right" style="width:78px" placeholder="续重" />
              </div>
            </template>
          </el-table-column>
          <el-table-column label="30kg以上 (首重+续重)" width="180" align="center">
            <template #default="{row}">
              <div class="zt-range">
                <el-input-number v-model="row.first_30up" :min="0" :step="0.1" :precision="2" size="small" controls-position="right" style="width:78px" placeholder="首重" />
                <span class="zt-plus">+</span>
                <el-input-number v-model="row.cont_30up" :min="0" :step="0.05" :precision="2" size="small" controls-position="right" style="width:78px" placeholder="续重" />
              </div>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <template #footer>
        <el-button @click="zoneTemplateDlgVisible = false">取消</el-button>
        <el-button type="primary" @click="confirmGenerateZoneRules" :loading="generatingZoneRules">生成规则</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="avgWeightDlgVisible" title="拉均重/偏差加价配置" width="600px" destroy-on-close>
      <div class="aw-help-box">
        <el-icon size="16" color="#409eff"><HelpFilled /></el-icon>
        <span>拉均重规则用于对平均重量偏低的客户进行加价惩罚，防止小件包裹过度占用快递资源。</span>
        <ul class="aw-help-list">
          <li><strong>计算逻辑：</strong>按客户分组计算平均重量，低于基准重量时按偏差步长加价，加价分摊到每个包裹。</li>
          <li><strong>优先级：</strong>客户专属规则 > 全局规则。</li>
          <li><strong>重量上限：</strong>超过设定重量的包裹不参与平均计算，也不会被加价。</li>
        </ul>
      </div>
      <el-form :model="awForm" label-width="120px" class="aw-form">
        <el-form-item label="作用范围">
          <el-radio-group v-model="awForm.scope_type" :disabled="!!avgWeightRule && avgWeightRule.scope_type==='customer'">
            <el-radio value="global">全局规则（所有客户共用）</el-radio>
            <el-radio value="customer">客户专属（仅当前客户）</el-radio>
          </el-radio-group>
          <div class="form-tip" v-if="awForm.scope_type==='customer'">
            客户专属规则优先级高于全局规则
          </div>
        </el-form-item>
        <el-form-item label="基准重量">
          <el-input-number v-model="awForm.base_weight" :min="0.01" :step="0.1" :precision="2" style="width:180px" />
          <span class="form-unit">kg</span>
          <div class="form-tip">平均重量低于此值时，触发偏差加价</div>
        </el-form-item>
        <el-form-item label="重量上限">
          <el-input-number v-model="awForm.weight_limit" :min="0" :step="0.5" :precision="1" style="width:180px" />
          <span class="form-unit">kg</span>
          <div class="form-tip">超过此重量的包裹不参与拉均重计算和加价，0表示不限制</div>
        </el-form-item>
        <el-form-item label="偏差步长">
          <el-input-number v-model="awForm.step_weight" :min="0.01" :step="0.05" :precision="2" style="width:180px" />
          <span class="form-unit">kg</span>
          <div class="form-tip">每低于基准多少公斤，加一次价</div>
        </el-form-item>
        <el-form-item label="每步加价">
          <el-input-number v-model="awForm.step_price" :min="0.01" :step="0.1" :precision="2" style="width:180px" />
          <span class="form-unit">元/件</span>
          <div class="form-tip">每个偏差步长，每件货物加价多少</div>
        </el-form-item>
        <el-form-item label="单件最高加价">
          <el-input-number v-model="awForm.max_markup" :min="0" :step="0.5" :precision="2" style="width:180px" />
          <span class="form-unit">元</span>
          <div class="form-tip">0 表示不限制</div>
        </el-form-item>
        <el-form-item label="取整方式">
          <el-radio-group v-model="awForm.round_mode">
            <el-radio value="ceil">向上取整</el-radio>
            <el-radio value="round">四舍五入</el-radio>
            <el-radio value="floor">向下取整</el-radio>
          </el-radio-group>
          <div class="form-tip">偏差重量 ÷ 步长 的取整方式</div>
        </el-form-item>
        <el-form-item label="启用状态">
          <el-switch v-model="awForm.is_enabled" :active-value="1" :inactive-value="0" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="awForm.remark" type="textarea" :rows="2" placeholder="选填" maxlength="100" show-word-limit />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="avgWeightDlgVisible = false">取消</el-button>
        <el-button type="primary" @click="saveAvgWeightRule" :loading="savingAvgWeight">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onBeforeUnmount, watch } from 'vue'
import { useAppStore, type FreightRule, type CustomerInfo, type GlobalRule, type ProvinceSurcharge, type AvgWeightRule } from '@/stores/app'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Search, Upload, Download, Setting, ArrowDown, UserFilled, Avatar, Close, Document, Grid, HelpFilled } from '@element-plus/icons-vue'

const store = useAppStore()

// ====== 全局规则 ======
const globalCollapsed = ref(false)
const savingGlobal = ref(false)
const grForm = reactive<GlobalRule>({
  default_first_weight: 1.0,
  default_first_price: 5.0,
  default_cont_price: 2.0,
  default_min_fee: 0,
  no_weight_price: 5.0,
  markup_fixed: 0,
  markup_percent: 0,
})

async function loadGlobalRules() {
  try {
    const g = await store.fetchGlobalRules()
    if (g && typeof g === 'object') {
      Object.assign(grForm, g)
    }
  } catch {}
}

async function saveGlobal() {
  savingGlobal.value = true
  try {
    const r = await store.saveGlobalRules({...grForm})
    if (r.ok) { ElMessage.success('全局规则已保存') } else { ElMessage.error('保存失败') }
  } catch { ElMessage.error('保存失败') }
  finally { savingGlobal.value = false }
}

// ====== 省份加价 ======
const provinceSurcharges = ref<ProvinceSurcharge[]>([])
const psForm = reactive({ id: 0, province_name: '', surcharge: 1, remark: '' })

async function loadProvinceSurcharges() {
  try {
    const data = await store.fetchProvinceSurcharges()
    provinceSurcharges.value = Array.isArray(data) ? data : []
  } catch {
    provinceSurcharges.value = []
  }
}

async function addProvinceSurcharge() {
  if (!psForm.province_name) return
  // 检查重复
  const exists = provinceSurcharges.value.find(p => p.province_name === psForm.province_name)
  if (exists) {
    ElMessage.warning('该省份已添加加价，请先删除再重新添加')
    return
  }
  const r = await store.saveProvinceSurcharge({ ...psForm })
  if (r.id) {
    ElMessage.success('已添加省份加价')
    psForm.province_name = ''
    psForm.surcharge = 1
    await loadProvinceSurcharges()
  }
}

async function removeProvinceSurcharge(id: number) {
  await store.deleteProvinceSurcharge(id)
  ElMessage.success('已删除')
  await loadProvinceSurcharges()
}

// ====== 客户列表 ======
const customers = ref<CustomerInfo[]>([])
const activeCustomer = ref('')
const customerSearch = ref('')
const loadingCustomers = ref(false)
const showAddCustomer = ref(false)
const newCustomerName = ref('')
const importing = ref(false)
const exporting = ref(false)

// ====== 拖拽复制 ======
// ⚠️ dragstart 时不能修改 reactive ref（会触发 Vue 重渲染 → DOM 突变 → 浏览器取消拖拽）
// 改为：1) dataTransfer 存数据 2) 原生 DOM 操作样式 3) 普通变量存源名
const dragOverTarget = ref('')
let dragSourceName = ''          // 非响应式，避免拖拽中 DOM 被替换
let nativeDropHandler: ((e: DragEvent) => void) | null = null
let nativeDragOverHandler: ((e: DragEvent) => void) | null = null

function onDragStart(e: DragEvent, name: string) {
  console.log('[onDragStart] 开始拖拽:', name)
  dragSourceName = name
  // 原生 DOM 加样式，不走 Vue 响应式
  const el = e.target as HTMLElement
  el.classList.add('drag-source')
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'copy'
    e.dataTransfer.setData('text/plain', name)
  }
}
function onDragEnd(e: DragEvent) {
  console.log('[onDragEnd] 拖拽结束, source:', dragSourceName)
  const el = e.target as HTMLElement
  el.classList.remove('drag-source')
  dragSourceName = ''
  dragOverTarget.value = ''
}

function onDragLeave(name: string) {
  if (dragOverTarget.value === name) {
    dragOverTarget.value = ''
  }
}

// 原生 dragover/drop — 挂 .customer-list 即可（dragstart 不再触发 DOM 突变，拖拽不会被取消）
function setupNativeDragDrop() {
  const listEl = document.querySelector('.customer-list')
  if (!listEl) return
  console.log('[native] 绑定 dragover/drop 到 .customer-list')

  nativeDragOverHandler = (e: DragEvent) => {
    e.preventDefault()
    const item = (e.target as HTMLElement).closest('.customer-item') as HTMLElement | null
    if (item) {
      const name = item.querySelector('.ci-name')?.textContent || ''
      if (name && name !== dragSourceName) {
        dragOverTarget.value = name
      }
    }
  }

  nativeDropHandler = (e: DragEvent) => {
    e.preventDefault()
    // ⚠️ getData 返回的是源名称（dragstart 设的），必须从 DOM 取目标元素名
    const item = (e.target as HTMLElement).closest('.customer-item') as HTMLElement | null
    if (!item) return
    const targetName = item.querySelector('.ci-name')?.textContent || ''
    if (targetName) handleDrop(targetName)
  }

  listEl.addEventListener('dragover', nativeDragOverHandler as EventListener)
  listEl.addEventListener('drop', nativeDropHandler as EventListener)
}

function teardownNativeDragDrop() {
  const listEl = document.querySelector('.customer-list')
  if (listEl) {
    if (nativeDragOverHandler) listEl.removeEventListener('dragover', nativeDragOverHandler as EventListener)
    if (nativeDropHandler) listEl.removeEventListener('drop', nativeDropHandler as EventListener)
  }
}

async function handleDrop(targetName: string) {
  const sourceName = dragSourceName
  console.log('[handleDrop] sourceName:', sourceName, 'targetName:', targetName)
  dragOverTarget.value = ''
  dragSourceName = ''
  if (!sourceName || sourceName === targetName) {
    console.log('[handleDrop] 跳过: sourceName为空或相同')
    return
  }
  if (!window.confirm(`将「${sourceName}」的全部规则复制到「${targetName}」？`)) {
    return
  }
  console.log('[handleDrop] 用户确认，开始复制...')
  try {
    const r = await store.copyCustomerRules(sourceName, targetName)
    console.log('[handleDrop] API 返回:', r)
    if (r.ok) {
      ElMessage.success(`已复制 ${r.count} 条规则到「${targetName}」`)
      await loadCustomers()
      if (activeCustomer.value === targetName) loadCustomerRules(targetName)
    } else {
      ElMessage.error(r.error || '复制失败')
    }
  } catch (e) {
    console.error('[handleDrop] 复制异常:', e)
    ElMessage.error('复制失败，请重试')
  }
}

const filteredCustomers = computed(() => {
  const list = customers.value || []
  if (!customerSearch.value) return list
  const s = customerSearch.value.toLowerCase()
  return list.filter(c => c.name && c.name.toLowerCase().includes(s))
})

async function loadCustomers() {
  loadingCustomers.value = true
  try {
    const data = await store.fetchCustomers()
    customers.value = Array.isArray(data) ? data : []
  } catch {
    customers.value = []
  } finally { loadingCustomers.value = false }
}

function selectCustomer(name: string) {
  if (activeCustomer.value === name) return
  activeCustomer.value = name
  loadCustomerRules(name)
}

async function confirmAddCustomer() {
  const name = newCustomerName.value.trim()
  if (!name) return
  showAddCustomer.value = false
  newCustomerName.value = ''
  
  try {
    ElMessage.info(`正在为客户「${name}」生成规则...`)
    activeCustomer.value = name
    
    const priceTable = await store.fetchSamplePriceTable()
    const result: any = await store.generateZoneRules(name, 'actual_weight', 'bracket', priceTable || {})
    
    if (result.ok) {
      ElMessage.success(`客户「${name}」已创建，共生成 ${result.count || 0} 条规则`)
      await loadCustomers()
      if (activeCustomer.value) {
        await loadCustomerRules(activeCustomer.value)
      }
    } else {
      ElMessage.error(result.msg || '创建客户失败')
      activeCustomer.value = ''
    }
  } catch (e) {
    console.error('新增客户异常:', e)
    ElMessage.error('创建客户失败')
    activeCustomer.value = ''
  }
}

async function handleDeleteCustomer(name: string) {
  await store.deleteCustomer(name)
  ElMessage.success(`已删除客户「${name}」及其所有规则`)
  if (activeCustomer.value === name) { activeCustomer.value = ''; customerRules.value = [] }
  await loadCustomers()
}

async function handleImport(uploadFile: any) {
  importing.value = true
  try {
    const r = await store.importCustomerRules(uploadFile.raw)
    if (r.ok) {
      ElMessage.success(`成功导入 ${r.count} 条规则`)
      await loadCustomers()
      // 如果有客户列表，选中第一个
      if (customers.value.length > 0) selectCustomer(customers.value[0].name)
    } else {
      ElMessage.error(r.error || '导入失败')
    }
  } catch { ElMessage.error('导入失败') }
  finally { importing.value = false }
}

async function handleExport() {
  exporting.value = true
  try {
    await store.exportCustomerRules(activeCustomer.value || '')
    ElMessage.success('导出成功')
  } catch { ElMessage.error('导出失败') }
  finally { exporting.value = false }
}

// 当客户列表更新时自动刷新 activeCustomer 的规则
watch(customers, async (list) => {
  if (activeCustomer.value) {
    // 检查当前客户是否还在列表中
    const found = list.find(c => c.name === activeCustomer.value)
    if (found) {
      loadCustomerRules(activeCustomer.value)
    }
  }
})

// ====== 右侧规则 ======
const customerRules = ref<FreightRule[]>([])
const loadingRules = ref(false)
const selectedRuleIds = ref<number[]>([])
const viewMode = ref<'zone' | 'list'>('zone')
const avgWeightRule = ref<AvgWeightRule | null>(null)
const avgWeightDlgVisible = ref(false)
const savingAvgWeight = ref(false)
const awForm = reactive<AvgWeightRule>({
  id: 0,
  scope_type: 'global',
  customer_name: '',
  base_weight: 0.3,
  weight_limit: 3,
  step_weight: 0.1,
  step_price: 0.1,
  max_markup: 0,
  round_mode: 'ceil',
  is_enabled: 0,
  remark: '',
})

const zoneGroups = computed(() => {
  const list = customerRules.value || []
  const bracketRules = list.filter(r => r.calc_mode === 'bracket' && r.zone_name)
  const groups: Record<string, { zone_name: string; zone_order: number; rules: FreightRule[] }> = {}
  for (const r of bracketRules) {
    if (!groups[r.zone_name]) {
      const zoneOrder = extractZoneOrder(r.zone_name)
      groups[r.zone_name] = { zone_name: r.zone_name, zone_order: zoneOrder, rules: [] }
    }
    groups[r.zone_name].rules.push(r)
  }
  return Object.values(groups).sort((a, b) => a.zone_order - b.zone_order)
})

const bracketRules = computed(() => {
  const list = customerRules.value || []
  return list.filter(r => r.calc_mode === 'bracket' && r.zone_name)
    .sort((a, b) => {
      const za = extractZoneOrder(a.zone_name)
      const zb = extractZoneOrder(b.zone_name)
      if (za !== zb) return za - zb
      return (a.province || '').localeCompare(b.province || '')
    })
})

function extractZoneOrder(name: string): number {
  const m = name.match(/(\d+)/)
  return m ? parseInt(m[1]) : 99
}

function getBracketFixedPrice(rule: FreightRule, from: number, to: number): string {
  if (!rule.brackets || rule.brackets.length === 0) return '—'
  const b = rule.brackets.find(x => x.weight_from === from && (x.weight_to === to || (to === 0 && x.weight_to === 0)))
  if (!b || b.calc_type !== 'fixed') return '—'
  return '¥' + b.fixed_price.toFixed(2)
}

function getBracketFirstPrice(rule: FreightRule, from: number, to: number): string {
  if (!rule.brackets || rule.brackets.length === 0) return '—'
  const b = rule.brackets.find(x => x.weight_from === from && (x.weight_to === to || (to === 0 && x.weight_to === 0)))
  if (!b || b.calc_type !== 'first_cont') return '—'
  return '¥' + b.first_price.toFixed(2)
}

function getBracketContPrice(rule: FreightRule, from: number, to: number): string {
  if (!rule.brackets || rule.brackets.length === 0) return '—'
  const b = rule.brackets.find(x => x.weight_from === from && (x.weight_to === to || (to === 0 && x.weight_to === 0)))
  if (!b || b.calc_type !== 'first_cont') return '—'
  return '¥' + b.cont_price.toFixed(2)
}

function getContModeLabel(mode: string): string {
  if (mode === 'actual_weight') return '实际重量'
  if (mode === 'hundred_gram') return '百克续'
  return '全续'
}

async function loadCustomerRules(name: string) {
  loadingRules.value = true
  try {
    const data = await store.fetchRulesByCustomer(name)
    customerRules.value = Array.isArray(data) ? data : []
  } catch {
    customerRules.value = []
  }
  // 加载拉均重规则
  try {
    const aw = await store.fetchAvgWeightRule(name)
    avgWeightRule.value = aw && aw.id ? aw : null
  } catch {
    avgWeightRule.value = null
  }
  finally { loadingRules.value = false; selectedRuleIds.value = [] }
}

async function toggleAvgWeightRule(enabled: boolean) {
  if (!avgWeightRule.value) return
  try {
    await store.toggleAvgWeight(avgWeightRule.value.id, enabled ? 1 : 0)
    avgWeightRule.value.is_enabled = enabled ? 1 : 0
    ElMessage.success(enabled ? '已启用拉均重' : '已停用拉均重')
  } catch (e) {
    ElMessage.error('操作失败')
  }
}

function openAvgWeightDlg() {
  if (avgWeightRule.value) {
    Object.assign(awForm, avgWeightRule.value)
  } else {
    awForm.id = 0
    awForm.scope_type = 'customer'
    awForm.customer_name = activeCustomer.value || ''
    awForm.base_weight = 0.3
    awForm.weight_limit = 3
    awForm.step_weight = 0.1
    awForm.step_price = 0.1
    awForm.max_markup = 0
    awForm.round_mode = 'ceil'
    awForm.is_enabled = 0
    awForm.remark = ''
  }
  avgWeightDlgVisible.value = true
}

async function saveAvgWeightRule() {
  if (awForm.base_weight <= 0) {
    ElMessage.warning('基准重量必须大于0')
    return
  }
  if (awForm.step_weight <= 0) {
    ElMessage.warning('偏差步长必须大于0')
    return
  }
  if (awForm.step_price <= 0) {
    ElMessage.warning('每步加价必须大于0')
    return
  }
  savingAvgWeight.value = true
  try {
    const saveData = { ...awForm }
    if (saveData.scope_type === 'customer') {
      saveData.customer_name = activeCustomer.value || ''
    } else {
      saveData.customer_name = ''
    }
    const res = await store.saveAvgWeightRule(saveData)
    if (res && res.id) {
      ElMessage.success('保存成功')
      avgWeightDlgVisible.value = false
      if (activeCustomer.value) {
        const aw = await store.fetchAvgWeightRule(activeCustomer.value)
        avgWeightRule.value = aw && aw.id ? aw : null
      }
    } else {
      ElMessage.error(res?.message || '保存失败')
    }
  } catch (e) {
    ElMessage.error('保存失败')
  } finally {
    savingAvgWeight.value = false
  }
}

async function handleDeleteRule(id: number) {
  try {
    const result = await store.deleteRule(id)
    if (result.ok !== false) {
      ElMessage.success('已删除')
      // 立即刷新当前客户规则列表
      if (activeCustomer.value) {
        await loadCustomerRules(activeCustomer.value)
      }
      await loadCustomers()
    } else {
      ElMessage.error(result.error || '删除失败')
    }
  } catch (e) {
    console.error('删除规则异常:', e)
    ElMessage.error('删除失败')
  }
}

async function batchDelete() {
  if (selectedRuleIds.value.length === 0) return
  await store.deleteRulesBatch(selectedRuleIds.value)
  ElMessage.success(`已删除 ${selectedRuleIds.value.length} 条规则`)
  await loadCustomers()
  if (activeCustomer.value) loadCustomerRules(activeCustomer.value)
}

async function toggleRule(row: FreightRule) {
  const newEnabled = row.is_enabled === 1 ? 0 : 1
  const oldEnabled = row.is_enabled
  row.is_enabled = newEnabled
  try {
    await store.saveRule({ 
      id: row.id,
      rule_type: row.rule_type,
      customer_name: row.customer_name,
      province: row.province,
      cont_mode: row.cont_mode,
      calc_mode: row.calc_mode,
      is_enabled: newEnabled,
    })
    ElMessage.success(newEnabled ? '已启用' : '已禁用')
    // 刷新当前客户的规则列表以确认保存成功
    if (activeCustomer.value) {
      await loadCustomerRules(activeCustomer.value)
    }
  } catch {
    row.is_enabled = oldEnabled
    ElMessage.error('操作失败')
  }
}

// ====== 规则弹窗 ======
const ruleDlgVisible = ref(false)
const editingRule = ref<FreightRule | null>(null)
const savingRule = ref(false)
const ruleForm = reactive({
  id: 0, rule_type: 'customer' as string, customer_name: '',
  province: '', calc_mode: 'bracket' as string, cont_mode: 'actual_weight' as string,
  first_weight: 1, first_price: 5, cont_price: 2,
  min_fee: 0, max_fee: 0, surcharge: 0,
  campaign_name: '', campaign_start: '', campaign_end: '',
  is_enabled: 1, remark: '',
})

function resetRuleForm() {
  Object.assign(ruleForm, {
    id: 0, rule_type: 'customer', customer_name: activeCustomer.value,
    province: '', calc_mode: 'bracket', cont_mode: 'actual_weight',
    first_weight: 1, first_price: 5, cont_price: 2,
    min_fee: 0, max_fee: 0, surcharge: 0,
    campaign_name: '', campaign_start: '', campaign_end: '',
    is_enabled: 1, remark: '',
  })
}

function openRuleDlg(rule: FreightRule | null) {
  if (rule) {
    editingRule.value = rule
    Object.assign(ruleForm, { ...rule })
  } else {
    editingRule.value = null
    resetRuleForm()
  }
  ruleDlgVisible.value = true
}

async function saveRule() {
  savingRule.value = true
  try {
    await store.saveRule({ ...ruleForm })
    ElMessage.success('保存成功')
    ruleDlgVisible.value = false
    // 刷新
    await loadCustomers()
    if (activeCustomer.value) loadCustomerRules(activeCustomer.value)
  } catch { ElMessage.error('保存失败') }
  finally { savingRule.value = false }
}

// ====== 区域模板生成 ======
const zoneTemplateDlgVisible = ref(false)
const generatingZoneRules = ref(false)
const zoneTemplates = ref<any[]>([])
const zonePriceList = ref<any[]>([])
const zoneForm = reactive({
  calc_mode: 'bracket',
  cont_mode: 'actual_weight',
})

async function openZoneTemplateDlg() {
  if (!activeCustomer.value) {
    ElMessage.warning('请先选择客户')
    return
  }
  zoneForm.calc_mode = 'bracket'
  zoneForm.cont_mode = 'actual_weight'
  zonePriceList.value = []
  zoneTemplateDlgVisible.value = true
  try {
    const templates = await store.fetchZoneTemplates()
    zoneTemplates.value = Array.isArray(templates) ? templates : []
    const sample = await store.fetchSamplePriceTable()
    buildZonePriceList(sample)
  } catch {}
}

function buildZonePriceList(sample: any) {
  const templates = zoneTemplates.value || []
  const priceMap = sample || {}
  zonePriceList.value = templates.map(t => {
    const p = priceMap[t.zone_name] || {}
    return {
      zone_name: t.zone_name,
      zone_order: t.zone_order,
      provinces: t.provinces || [],
      price_0_05: p.price_0_05 || 0,
      price_05_1: p.price_05_1 || 0,
      price_1_2: p.price_1_2 || 0,
      price_2_3: p.price_2_3 || 0,
      first_3_30: p.first_3_30 || 0,
      cont_3_30: p.cont_3_30 || 0,
      first_30up: p.first_30up || 0,
      cont_30up: p.cont_30up || 0,
    }
  })
}

async function loadSamplePrice() {
  try {
    const sample = await store.fetchSamplePriceTable()
    buildZonePriceList(sample)
    ElMessage.success('已加载参考价格')
  } catch {
    ElMessage.error('加载失败')
  }
}

function zoneTagType(order: number): string {
  const types: Record<number, string> = {
    1: 'success', 2: 'primary', 3: 'warning', 4: 'danger', 5: 'info', 6: 'warning',
  }
  return types[order] || 'info'
}

async function confirmGenerateZoneRules() {
  if (!activeCustomer.value) return
  try {
    await ElMessageBox.confirm(
      `将为客户「${activeCustomer.value}」生成 ${zonePriceList.value.length} 个区域的区间计费规则。\n注意：该客户已有的区域型/区间型规则将被删除，确定继续吗？`,
      '确认生成',
      { type: 'warning', confirmButtonText: '确定生成', cancelButtonText: '取消' }
    )
  } catch { return }

  generatingZoneRules.value = true
  try {
    const priceTable: Record<string, any> = {}
    for (const row of zonePriceList.value) {
      priceTable[row.zone_name] = {
        price_0_05: Number(row.price_0_05) || 0,
        price_05_1: Number(row.price_05_1) || 0,
        price_1_2: Number(row.price_1_2) || 0,
        price_2_3: Number(row.price_2_3) || 0,
        first_3_30: Number(row.first_3_30) || 0,
        cont_3_30: Number(row.cont_3_30) || 0,
        first_30up: Number(row.first_30up) || 0,
        cont_30up: Number(row.cont_30up) || 0,
      }
    }
    const result: any = await store.generateZoneRules(activeCustomer.value, zoneForm.cont_mode, zoneForm.calc_mode, priceTable)
    if (result.ok) {
      ElMessage.success(result.msg || '生成成功')
      zoneTemplateDlgVisible.value = false
      await loadCustomers()
      if (activeCustomer.value) loadCustomerRules(activeCustomer.value)
    } else {
      ElMessage.error(result.msg || '生成失败')
    }
  } catch {
    ElMessage.error('生成失败')
  }
  finally { generatingZoneRules.value = false }
}

// ====== 省份列表 ======
const PROVINCES = [
  '北京','天津','上海','重庆',
  '河北','山西','辽宁','吉林','黑龙江',
  '江苏','浙江','安徽','福建','江西','山东',
  '河南','湖北','湖南','广东',
  '四川','贵州','云南','陕西','甘肃','青海',
  '广西','内蒙古','宁夏','新疆','西藏','海南',
  '香港','澳门','台湾',
]

onMounted(async () => {
  try { await loadGlobalRules() } catch {}
  try { await loadProvinceSurcharges() } catch {}
  try {
    await loadCustomers()
    if (Array.isArray(customers.value) && customers.value.length > 0) {
      selectCustomer(customers.value[0].name)
    }
  } catch {}
  // 用原生 addEventListener 绑定拖放（绕过 Vue 事件系统）
  try { setupNativeDragDrop() } catch {}
})

onBeforeUnmount(() => {
  teardownNativeDragDrop()
})
</script>

<style scoped>
.rules-page { display: flex; flex-direction: column; gap: 12px; max-width: 1400px; }

/* ====== 全局规则面板 ====== */
.global-panel {
  background: #fff; border-radius: 10px; border: 1px solid #e4e7ed;
  box-shadow: 0 1px 4px rgba(0,0,0,.04);
  overflow: hidden; transition: all .3s;
}
.global-panel.collapsed { border-color: #ebeef5; }
.global-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 16px; cursor: pointer; user-select: none;
  background: linear-gradient(135deg, #fafbff, #f5f7ff);
  border-bottom: 1px solid #ebeef5;
}
.global-panel.collapsed .global-header { border-bottom: none; }
.global-title { display: flex; align-items: center; gap: 8px; font-weight: 600; color: #303133; font-size: 14px; }
.collapse-icon { color: #909399; transition: transform .3s; }
.collapse-icon.rotated { transform: rotate(180deg); }
.global-body { padding: 16px; display: flex; flex-direction: column; gap: 14px; }
.global-row { display: flex; align-items: center; gap: 16px; }
.global-row-label { min-width: 80px; }
.global-fields { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.global-fields label { font-size: 13px; color: #606266; white-space: nowrap; }
.global-fields label em { font-size: 11px; color: #909399; font-style: normal; }
.global-fields :deep(.el-input-number) { width: 120px; }
.gr-desc { font-size: 12px; color: #909399; margin-left: 8px; font-style: italic; }
.global-actions { display: flex; justify-content: flex-end; padding-top: 4px; border-top: 1px dashed #ebeef5; }

/* ====== 左右分栏 ====== */
.rules-main { display: flex; gap: 12px; flex: 1; min-height: 500px; }

/* 左面板 */
.left-panel {
  width: 300px; min-width: 280px; background: #fff;
  border-radius: 10px; border: 1px solid #e4e7ed;
  box-shadow: 0 1px 4px rgba(0,0,0,.04);
  display: flex; flex-direction: column; overflow: hidden;
}
.left-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 14px; border-bottom: 1px solid #ebeef5;
  background: linear-gradient(135deg, #fafbff, #f5f7ff);
}
.left-header h4 { margin: 0; font-size: 14px; display: flex; align-items: center; gap: 6px; color: #303133; }
.left-search { padding: 10px 14px 6px; }
.left-actions { display: flex; gap: 8px; padding: 0 14px 10px; }
.left-actions .el-button { flex: 1; font-size: 12px; }
.customer-list { flex: 1; overflow-y: auto; padding: 4px 14px 14px; }
.customer-item {
  display: flex; align-items: center; justify-content: space-between;
  padding: 10px 12px; margin: 3px 0; border-radius: 8px;
  cursor: pointer; transition: all .2s; border: 1px solid transparent;
}
.customer-item:hover { background: #f5f7ff; border-color: #dce1f5; }
.customer-item.active {
  background: linear-gradient(135deg, #e8ecff, #e0e7ff);
  border-color: #a5b4fc;
  box-shadow: 0 0 0 1px #818cf8;
}
.ci-left { display: flex; align-items: center; gap: 8px; overflow: hidden; }
.ci-left .el-icon { color: #6366f1; flex-shrink: 0; }
.ci-name { font-size: 14px; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.ci-right { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.ci-del { color: #c0c4cc; cursor: pointer; transition: color .2s; }
.ci-del:hover { color: #f56c6c; }
/* 拖拽状态 */
.drag-source { opacity: 0.4; }
.drop-over {
  border-color: #6366f1 !important;
  background: linear-gradient(135deg, #e8ecff, #ddd6fe) !important;
  box-shadow: 0 0 0 2px #818cf8 !important;
}
.drag-hint-tip {
  text-align: center; padding: 6px; margin: 4px 14px; border-radius: 6px;
  background: #f0f5ff; color: #6366f1; font-size: 12px;
  border: 1px dashed #a5b4fc; animation: pulse-hint 1.5s infinite;
}
@keyframes pulse-hint {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}
.empty-customers { text-align: center; padding: 40px 0; color: #c0c4cc; font-size: 13px; }

/* 右面板 */
.right-panel {
  flex: 1; background: #fff; border-radius: 10px;
  border: 1px solid #e4e7ed; box-shadow: 0 1px 4px rgba(0,0,0,.04);
  display: flex; flex-direction: column; overflow: hidden; min-width: 0;
}
.right-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px; border-bottom: 1px solid #ebeef5;
  background: linear-gradient(135deg, #fafbff, #f5f7ff);
  flex-wrap: wrap; gap: 8px;
}
.rh-left { display: flex; align-items: center; gap: 10px; }
.rh-customer { font-size: 16px; font-weight: 700; color: #303133; }
.rh-actions { display: flex; align-items: center; gap: 8px; }
.rule-table-wrap { flex: 1; overflow: auto; padding: 12px 16px; }
.unit { color: #909399; font-size: 11px; }
.prov-all { font-weight: 600; color: #409eff; }
.bracket-prices { font-size: 12px; color: #606266; }
.empty-text { color: #c0c4cc; }
.no-customer {
  flex: 1; display: flex; flex-direction: column; align-items: center;
  justify-content: center; gap: 8px; color: #c0c4cc;
}
.no-customer .sub { font-size: 12px; color: #dcdfe6; }

/* ====== 省份加价 ====== */
.province-surcharge-row { align-items: flex-start; }
.province-surcharge-area { flex: 1; display: flex; flex-direction: column; gap: 8px; }
.ps-list { display: flex; flex-wrap: wrap; gap: 6px; }
.ps-tag { font-size: 13px; }
.ps-add { display: flex; align-items: center; gap: 8px; }
.ps-unit { font-size: 12px; color: #909399; }

/* ====== 区域模板弹窗 ====== */
.zt-info { margin-bottom: 12px; }
.zt-info b { color: #409eff; }
.zt-toolbar {
  display: flex; align-items: center; justify-content: space-between;
  margin-bottom: 12px; padding: 8px 12px; background: #fafafa;
  border-radius: 6px; border: 1px solid #ebeef5;
}
.zt-left { display: flex; align-items: center; gap: 8px; }
.zt-label { font-size: 13px; color: #606266; }
.zt-table-wrap { max-height: 420px; overflow-y: auto; }
.zt-table :deep(.el-table__cell) { padding: 6px 8px; }
.zt-provinces { font-size: 12px; color: #606266; line-height: 1.4; }
.zt-range { display: flex; align-items: center; justify-content: center; gap: 4px; }
.zt-plus { color: #909399; font-size: 12px; }

/* ====== 阶段报价视图 ====== */
.zone-view-wrap {
  max-height: calc(100vh - 420px); overflow-y: auto; padding-right: 4px;
}
.zone-table-container { width: 100%; }
.zone-big-table .price-fix {
  font-variant-numeric: tabular-nums;
  color: #303133;
}
.zone-big-table .price-first {
  font-variant-numeric: tabular-nums;
  color: #409eff;
}
.zone-big-table .price-cont {
  font-variant-numeric: tabular-nums;
  color: #67c23a;
}
.zone-big-table .rule-disabled { opacity: 0.45; }
.avgweight-cell {
  display: flex; flex-direction: column; gap: 6px; align-items: center;
  line-height: 1.4;
}
.aw-top {
  display: flex; align-items: center; gap: 8px;
  font-size: 12px; color: #606266;
}
.aw-base { font-weight: 600; color: #409eff; }
.aw-bottom {
  font-size: 12px; color: #909399;
}
.aw-step { color: #e6a23c; font-weight: 500; }
.aw-sep { margin: 0 2px; }
.aw-price { color: #67c23a; font-weight: 500; }
.aw-empty { color: #c0c4cc; }
.aw-edit {
  display: flex; align-items: center; gap: 6px;
  font-size: 12px;
}
.aw-scope-tag {
  font-size: 11px; padding: 1px 6px; border-radius: 10px;
  background: #ecf5ff; color: #409eff;
}
.aw-scope-tag.aw-customer {
  background: #f0f9eb; color: #67c23a;
}
.aw-empty-wrap {
  display: flex; align-items: center; justify-content: center;
}
.aw-help-box {
  background: #f0f5ff; border: 1px solid #e6f0ff; border-radius: 8px;
  padding: 12px 14px; margin-bottom: 16px; display: flex; flex-direction: column; gap: 8px;
}
.aw-help-box span { font-size: 13px; color: #606266; }
.aw-help-list { margin: 0; padding-left: 20px; font-size: 12px; color: #808080; }
.aw-help-list li { margin-bottom: 4px; line-height: 1.5; }
.aw-form .form-tip {
  font-size: 12px; color: #909399; margin-top: 4px;
}
.aw-form .form-unit {
  margin-left: 8px; font-size: 13px; color: #606266;
}
.empty-zone {
  display: flex; align-items: center; justify-content: center;
  padding: 40px 0;
}
</style>
