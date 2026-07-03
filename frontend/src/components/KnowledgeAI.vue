<template>
  <div class="kai-ai-wrap">
    <!-- 浮动猴子图标 -->
    <div
      class="monkey-float"
      :style="{ left: pos.x + 'px', top: pos.y + 'px' }"
      @mousedown="startDrag"
      @touchstart="startDrag"
      @click="handleIconClick"
    >
      <div class="monkey-icon">
        <img src="/monkey-icon.png" alt="AI助手" />
      </div>
      <div class="monkey-bubble" v-if="showBubble">
        <span>有问题问我呀~</span>
      </div>
    </div>

    <!-- 问答面板 -->
    <transition name="ai-panel">
      <div v-if="panelOpen" class="ai-panel" :style="panelStyle">
        <div class="ai-panel-header" @mousedown="startDragPanel" @touchstart="startDragPanel">
          <div class="ai-title">
            <img src="/monkey-icon.png" class="ai-title-icon" />
            <span>喵喵知识库</span>
          </div>
          <div class="ai-actions">
            <el-icon class="ai-action-btn" @click.stop="clearChat" title="清空对话"><Delete /></el-icon>
            <el-icon class="ai-action-btn close-btn" @click.stop="closePanel" title="关闭"><Close /></el-icon>
          </div>
        </div>

        <div class="ai-panel-body" ref="chatBodyRef">
          <div class="msg-wrap">
            <div v-for="(msg, idx) in messages" :key="idx" :class="['msg-item', msg.role]">
              <div v-if="msg.role === 'ai'" class="msg-avatar">
                <img src="/monkey-icon.png" />
              </div>
              <div class="msg-bubble">
                <div class="msg-content" v-html="msg.content"></div>
                <div v-if="msg.suggestions && msg.suggestions.length" class="msg-suggestions">
                  <span
                    v-for="(s, si) in msg.suggestions"
                    :key="si"
                    class="sug-item"
                    @click="sendMsg(s)"
                  >{{ s }}</span>
                </div>
              </div>
              <div v-if="msg.role === 'user'" class="msg-avatar user">
                <el-icon><UserFilled /></el-icon>
              </div>
            </div>
            <div v-if="thinking" class="msg-item ai">
              <div class="msg-avatar"><img src="/monkey-icon.png" /></div>
              <div class="msg-bubble thinking">
                <span class="dot"></span><span class="dot"></span><span class="dot"></span>
              </div>
            </div>
          </div>
        </div>

        <div class="ai-panel-footer">
          <div class="quick-questions">
            <span v-for="(q, i) in quickQuestions" :key="i" class="quick-q" @click="sendMsg(q)">{{ q }}</span>
          </div>
          <div class="input-row">
            <el-input
              v-model="inputText"
              placeholder="输入你的问题..."
              size="default"
              clearable
              @keyup.enter="handleSend"
            >
              <template #append>
                <el-button @click="handleSend">
                  <el-icon><Promotion /></el-icon>
                </el-button>
              </template>
            </el-input>
          </div>
        </div>
      </div>
    </transition>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, computed, nextTick, onMounted } from 'vue'
import { Close, Delete, UserFilled, Promotion } from '@element-plus/icons-vue'

const panelOpen = ref(false)
const inputText = ref('')
const thinking = ref(false)
const showBubble = ref(true)
const chatBodyRef = ref<HTMLElement>()

const pos = reactive({ x: 20, y: 300 })
const panelPos = reactive({ x: 100, y: 150 })

let dragging = false
let dragTarget = ''
let startX = 0, startY = 0, startPosX = 0, startPosY = 0
let movedDuringDrag = false

const messages = ref<Array<{role: string, content: string, suggestions?: string[]}>>([])

const quickQuestions = [
  '运费怎么算的',
  '阶梯定价是什么',
  '拉均重怎么用',
  '区域规则怎么配',
  '全局规则优先级',
]

const panelStyle = computed(() => ({
  left: panelPos.x + 'px',
  top: panelPos.y + 'px',
}))

// ============ 知识库 ============
interface KBItem {
  keywords: string[]
  title: string
  answer: string
  related?: string[]
}

const knowledgeBase: KBItem[] = [
  {
    keywords: ['运费计算', '怎么算', '计算规则', '计费方式', '运费怎么算', '计算公式', '计算流程'],
    title: '运费计算完整流程',
    answer: `<p><strong>运费计算优先级（从高到低匹配规则）：</strong></p>
<ol>
  <li><strong>客户专属规则</strong> - 匹配客户名+省份，优先级最高</li>
  <li><strong>活动规则</strong> - 同一客户下，在有效期内的活动价</li>
  <li><strong>全局规则</strong> - 所有客户通用的全局定价</li>
  <li><strong>保底规则</strong> - 都不匹配时用全局默认价</li>
  <li><strong>兜底5元</strong> - 连全局规则都没有时的默认值</li>
</ol>
<p><strong>完整计算步骤：</strong></p>
<ol>
  <li>找匹配的规则（按上面优先级）</li>
  <li>按计费模式算基础运费（标准/阶梯）</li>
  <li>加偏远附加费（surcharge）</li>
  <li>加省份加价（全局省份加价）</li>
  <li>应用保底价/最高价（规则自身的）</li>
  <li>应用全局加价（固定 + 百分比）</li>
  <li>保留2位小数</li>
</ol>
<p><strong>零重量保护：</strong>重量为0时用全局的 no_weight_price（零重价格）</p>`,
    related: ['标准计费模式', '阶梯计费模式', '续重模式', '保底规则', '全局加价'],
  },
  {
    keywords: ['标准', 'simple', '首重续重', '首重', '续重', '传统计费', '标准模式'],
    title: '标准计费模式（首重续重）',
    answer: `<p><strong>标准模式</strong>就是最常见的首重+续重计费方式。</p>
<p><strong>计算公式：</strong></p>
<pre>重量 ≤ 首重 → 运费 = 首重价
重量 > 首重 → 运费 = 首重价 + 续重单价 × 续重单位数</pre>
<p><strong>续重单位数取决于续重模式：</strong></p>
<ul>
  <li><strong>整kg续重（full_kg）</strong>：向上取整到整kg，如1.2kg算2kg</li>
  <li><strong>实际重量（actual_weight）</strong>：按实际重量乘，如1.2kg就是1.2</li>
  <li><strong>百克续重（hundred_gram）</strong>：每100g为单位向上取整，如1.23kg算13个单位</li>
</ul>
<p><strong>例子：</strong>首重1kg=5元，续重2元/kg，重量2.3kg</p>
<ul>
  <li>整kg续重：5 + ceil(1.3)×2 = 5 + 2×2 = <strong>9元</strong></li>
  <li>实际重量：5 + 1.3×2 = <strong>7.6元</strong></li>
  <li>百克续重：5 + ceil(1.3×10)×(2/10) = 5 + 13×0.2 = <strong>7.6元</strong></li>
</ul>`,
    related: ['阶梯计费模式', '续重模式', '保底规则'],
  },
  {
    keywords: ['阶梯', 'bracket', '区间', '阶段报价', '阶梯定价', '重量区间', '区间计费'],
    title: '阶梯计费模式（区间计费）',
    answer: `<p><strong>阶梯模式</strong>把重量分成多个区间，每个区间有独立的价格。</p>
<p><strong>两种区间类型：</strong></p>
<ul>
  <li><strong>一口价（fixed）</strong>：落在这个区间内固定价格，不管具体多少</li>
  <li><strong>首续重（first_cont）</strong>：以区间起始重量为首重，超出部分按续重算</li>
</ul>
<p><strong>系统默认6个区间：</strong></p>
<ol>
  <li>0 - 0.5kg：一口价</li>
  <li>0.5 - 1kg：一口价</li>
  <li>1 - 2kg：一口价</li>
  <li>2 - 3kg：一口价</li>
  <li>3 - 30kg：首续重（首重3kg）</li>
  <li>30kg以上：首续重（首重3kg，weight_to=0表示无上限）</li>
</ol>
<p><strong>重要说明：</strong></p>
<ul>
  <li>每个区间可以设置自己的续重模式，不设置就用规则的</li>
  <li>没有找到匹配区间时，自动降级到标准模式计算</li>
  <li>区间数据和规则绑定，通过「区域模板生成」批量设置</li>
</ul>`,
    related: ['标准计费模式', '区域模板生成', '续重模式'],
  },
  {
    keywords: ['续重模式', 'full_kg', 'actual_weight', 'hundred_gram', '整kg', '实际重量', '百克'],
    title: '续重模式详解',
    answer: `<p><strong>三种续重模式对比：</strong></p>
<table style="width:100%;border-collapse:collapse;font-size:12px;">
<tr><th style="border:1px solid #ddd;padding:4px;">模式</th><th style="border:1px solid #ddd;padding:4px;">说明</th><th style="border:1px solid #ddd;padding:4px;">1.2kg续重</th></tr>
<tr><td style="border:1px solid #ddd;padding:4px;">整kg续重</td><td style="border:1px solid #ddd;padding:4px;">不足1kg按1kg算，向上取整</td><td style="border:1px solid #ddd;padding:4px;">2kg</td></tr>
<tr><td style="border:1px solid #ddd;padding:4px;">实际重量</td><td style="border:1px solid #ddd;padding:4px;">按实际重量精确计算</td><td style="border:1px solid #ddd;padding:4px;">1.2kg</td></tr>
<tr><td style="border:1px solid #ddd;padding:4px;">百克续重</td><td style="border:1px solid #ddd;padding:4px;">每100g为单位向上取整</td><td style="border:1px solid #ddd;padding:4px;">12个单位</td></tr>
</table>
<p><strong>计算方式：</strong></p>
<ul>
  <li><strong>整kg续重</strong>：ceil(超出重量) × 续重单价</li>
  <li><strong>实际重量</strong>：超出重量 × 续重单价</li>
  <li><strong>百克续重</strong>：ceil(超出重量 × 10) × (续重单价 / 10)</li>
</ul>
<p>标准模式和阶梯模式都可以设置续重模式。</p>`,
    related: ['标准计费模式', '阶梯计费模式'],
  },
  {
    keywords: ['拉均重', '偏差加价', '平均重量', '均重', 'avg_weight', '重量偏差'],
    title: '拉均重/偏差加价',
    answer: `<p><strong>拉均重</strong>是按客户包裹的平均重量来额外加价的功能，按批次整体计算。</p>
<p><strong>触发条件：</strong>批次平均重量 > 基准重量</p>
<p><strong>计算公式：</strong></p>
<pre>偏差 = 平均重量 - 基准重量
每件加价 = 偏差 × 每公斤加价
（超过单件最高加价时取上限）</pre>
<p><strong>配置项说明：</strong></p>
<ul>
  <li><strong>作用范围</strong>：全局（所有客户）/ 客户专属（优先级更高）</li>
  <li><strong>基准重量</strong>：平均超过这个值才加价</li>
  <li><strong>重量上限</strong>：超过此重量的大包裹不参与计算也不加价（0=不限制）</li>
  <li><strong>每公斤加价</strong>：每超1kg每件加多少钱</li>
  <li><strong>单件最高加价</strong>：每件最多加多少（0=不限制）</li>
</ul>
<p><strong>完整流程：</strong></p>
<ol>
  <li>按客户分组</li>
  <li>排除超过重量上限的包裹</li>
  <li>计算剩余包裹的平均重量</li>
  <li>平均>基准？→ 计算每件加价</li>
  <li>参与的每件包裹都加上同样的金额</li>
</ol>
<p><strong>例子：</strong>基准0.5kg，每公斤加2元。5个包裹（0.3、0.6、0.8、1.0、4.0kg），上限3kg</p>
<p>排除4.0kg，剩4个平均0.675kg，偏差0.175kg，每件加 0.175×2 = 0.35元</p>`,
    related: ['运费计算完整流程', '全局加价', '省份加价'],
  },
  {
    keywords: ['区域规则', '分区', '六区', '6区', '一区二区', '区域模板', 'zone', '区域划分'],
    title: '区域规则与六区体系',
    answer: `<p><strong>系统采用6区体系</strong>，按地理位置远近把省份分成6个区域，每个区域一套价格。</p>
<p><strong>六区划分：</strong></p>
<ul>
  <li><strong>一区</strong>：江浙沪皖等最近的省份</li>
  <li><strong>二区</strong>：安徽、福建、江西、山东、广东等</li>
  <li><strong>三区</strong>：北京、天津、河北、河南、湖北、湖南等</li>
  <li><strong>四区</strong>：山西、辽宁、吉林、黑龙江、陕西、四川、重庆等</li>
  <li><strong>五区</strong>：云南、贵州、甘肃、青海、宁夏、广西、内蒙古等</li>
  <li><strong>六区</strong>：新疆、西藏、海南、香港、澳门、台湾</li>
</ul>
<p><strong>区域模板生成：</strong></p>
<ul>
  <li>设置好每个区的价格方案（首重、续重、各阶梯价等）</li>
  <li>一键生成该客户所有省份的规则</li>
  <li>生成前会删除该客户已有的区域型规则（避免重复）</li>
  <li>非区域型规则（全国通配、自定义省份等）不会被删</li>
</ul>`,
    related: ['阶梯计费模式', '区域模板生成', '客户管理'],
  },
  {
    keywords: ['全局规则', '保底', '默认规则', '全局加价', '保底规则', 'fallback', 'default'],
    title: '全局规则与保底规则',
    answer: `<p><strong>全局规则</strong>是所有客户共用的配置，包含默认定价和加价设置。</p>
<p><strong>全局规则内容：</strong></p>
<ul>
  <li><strong>默认首重/续重</strong>：保底用的标准价格</li>
  <li><strong>默认保底价</strong>：保底规则的最低收费</li>
  <li><strong>零重价格</strong>：重量为0时的特殊价格</li>
  <li><strong>全局固定加价</strong>：每票加固定金额</li>
  <li><strong>全局百分比加价</strong>：运费 × 百分比（四舍五入到分）</li>
</ul>
<p><strong>保底规则什么时候用？</strong></p>
<p>当某个客户+省份找不到任何匹配的规则（客户规则、活动规则、全局规则都没有）时，就用全局默认的保底价格。</p>
<p><strong>保底规则长什么样？</strong></p>
<ul>
  <li>rule_type = 'default'，系统自动创建</li>
  <li>不能删除，只能修改价格</li>
  <li>是所有计算的最后一道防线</li>
</ul>
<p>连全局规则都没有配置时，兜底返回5元。</p>`,
    related: ['运费计算完整流程', '全局加价', '省份加价'],
  },
  {
    keywords: ['全局加价', '固定加价', '百分比加价', 'markup'],
    title: '全局加价',
    answer: `<p><strong>全局加价</strong>是在计算完基础运费后，统一加上的费用。</p>
<p><strong>两种加价方式可以同时生效：</strong></p>
<ul>
  <li><strong>固定加价</strong>：每票直接加固定金额</li>
  <li><strong>百分比加价</strong>：基础运费 × 百分比</li>
</ul>
<p><strong>计算公式：</strong></p>
<pre>加价金额 = 固定加价 + round(基础运费 × 百分比)
最终运费 = 基础运费 + 加价金额</pre>
<p><strong>注意：</strong></p>
<ul>
  <li>百分比加价是按基础运费（含偏远附加费、省份加价、保底价后）来算的</li>
  <li>百分比四舍五入到分</li>
  <li>全局加价在保底价/最高价之后应用</li>
</ul>
<p><strong>计算顺序回顾：</strong>基础运费 → 偏远附加费 → 省份加价 → 保底价/最高价 → 全局加价</p>`,
    related: ['省份加价', '保底规则', '运费计算完整流程'],
  },
  {
    keywords: ['省份加价', 'province_surcharge', '省价', '省份附加费'],
    title: '省份加价',
    answer: `<p><strong>省份加价</strong>是按目的省份每票加收的费用，在全局规则里配置。</p>
<p><strong>添加方式：</strong></p>
<ul>
  <li>在「系统设置」→「全局规则」里添加</li>
  <li>选择省份 + 设置加价金额</li>
  <li>可以添加多个省份，每个省份独立设置</li>
  <li>系统会自动去重，同一个省份不会重复加</li>
</ul>
<p><strong>计算时机：</strong></p>
<p>基础运费 + 偏远附加费 → <strong>省份加价</strong> → 保底价/最高价 → 全局加价</p>
<p><strong>和偏远附加费的区别：</strong></p>
<ul>
  <li><strong>偏远附加费（surcharge）</strong>：是每条规则自己设置的，不同规则可以不同</li>
  <li><strong>省份加价</strong>：是全局统一的，所有规则命中后都加</li>
</ul>`,
    related: ['全局加价', '保底规则', '运费计算完整流程'],
  },
  {
    keywords: ['保底价', '最低价', 'min_fee', '最高价', 'max_fee', '封顶'],
    title: '保底价与最高价',
    answer: `<p><strong>保底价（min_fee）</strong>：运费低于这个价按保底价收，相当于最低消费</p>
<p><strong>最高价（max_fee）</strong>：运费高于这个价按最高价收，相当于封顶价</p>
<p><strong>应用时机：</strong></p>
<p>基础运费 + 偏远附加费 + 省份加价 → <strong>保底价/最高价</strong> → 全局加价</p>
<p><strong>优先级：</strong></p>
<ul>
  <li>每条规则可以设置自己的 min_fee / max_fee</li>
  <li>规则自身的保底价 优先于 全局保底</li>
  <li>全局保底只在没有匹配规则时才用</li>
</ul>
<p><strong>例子：</strong>算出来运费3元，保底价5元 → 按5元收</p>
<p><strong>例子：</strong>算出来运费100元，最高价80元 → 按80元收</p>
<p>设置为0表示不限制。</p>`,
    related: ['全局规则', '运费计算完整流程'],
  },
  {
    keywords: ['偏远附加费', 'surcharge', '附加费', '偏远费'],
    title: '偏远附加费',
    answer: `<p><strong>偏远附加费（surcharge）</strong>是每条规则可以单独设置的额外费用。</p>
<p><strong>特点：</strong></p>
<ul>
  <li>每条规则独立设置，不同省份/客户可以不同</li>
  <li>固定金额，每票加一次</li>
  <li>在基础运费之后直接加上</li>
</ul>
<p><strong>计算位置：</strong></p>
<p>基础运费 → <strong>偏远附加费</strong> → 省份加价 → 保底价/最高价 → 全局加价</p>
<p><strong>和省份加价的区别：</strong></p>
<ul>
  <li><strong>偏远附加费</strong>：规则级，灵活设置，每条规则可以不一样</li>
  <li><strong>省份加价</strong>：全局级，统一管理，所有规则命中都加</li>
</ul>`,
    related: ['省份加价', '全局加价', '运费计算完整流程'],
  },
  {
    keywords: ['客户管理', '新增客户', '删除客户', '客户列表', '添加客户'],
    title: '客户管理',
    answer: `<p><strong>客户列表</strong>在规则管理页面的左侧，展示所有有规则的客户。</p>
<p><strong>新增客户：</strong></p>
<ul>
  <li>点击「+ 新增客户」按钮</li>
  <li>输入客户名称，确认</li>
  <li>系统会自动为该客户生成6个区域的默认规则（基于区域模板）</li>
  <li>新增后可以直接在右侧编辑该客户的规则</li>
</ul>
<p><strong>删除客户：</strong></p>
<ul>
  <li>在客户列表里点客户名右边的删除按钮</li>
  <li>会删除该客户的所有规则（客户规则 + 活动规则）</li>
  <li>删除后客户从列表消失</li>
  <li>不可恢复，请谨慎操作</li>
</ul>
<p><strong>复制客户规则：</strong></p>
<ul>
  <li>拖拽一个客户到另一个客户上</li>
  <li>会把源客户的所有规则复制给目标客户</li>
  <li>目标客户已有规则不会被删除，是追加</li>
  <li>拉均重规则不会被复制</li>
</ul>
<p><strong>客户搜索：</strong>顶部搜索框可以按客户名筛选</p>`,
    related: ['规则导入导出', '区域模板生成'],
  },
  {
    keywords: ['规则管理', '新增规则', '编辑规则', '规则列表', '阶段报价', '列表视图'],
    title: '规则管理',
    answer: `<p><strong>规则管理页面</strong>有两种视图：</p>
<ul>
  <li><strong>阶段报价视图</strong>：按区域分组展示，适合批量管理阶梯定价</li>
  <li><strong>列表视图</strong>：所有规则平铺展示，适合查找和批量操作</li>
</ul>
<p><strong>规则的基本字段：</strong></p>
<ul>
  <li><strong>客户名称</strong>：所属客户</li>
  <li><strong>省份</strong>：目的省份，空=全国通配</li>
  <li><strong>计费模式</strong>：standard（标准）/ bracket（阶梯）</li>
  <li><strong>续重模式</strong>：full_kg / actual_weight / hundred_gram</li>
  <li><strong>首重/首重价/续重价</strong>：标准模式用</li>
  <li><strong>保底价/最高价</strong>：最低/最高收费</li>
  <li><strong>附加费</strong>：偏远附加费</li>
  <li><strong>规则类型</strong>：customer（客户规则）/ campaign（活动规则）</li>
  <li><strong>区域</strong>：所属区域（区域型规则才有）</li>
  <li><strong>启用状态</strong>：1=启用，0=禁用</li>
  <li><strong>备注</strong>：自定义说明</li>
</ul>
<p><strong>活动规则额外字段：</strong>活动名称、活动开始时间、活动结束时间</p>`,
    related: ['规则启用禁用', '删除规则', '活动规则'],
  },
  {
    keywords: ['活动规则', 'campaign', '活动价', '促销', '限时'],
    title: '活动规则',
    answer: `<p><strong>活动规则</strong>是有有效期的特殊规则，在有效期内优先级高于普通客户规则。</p>
<p><strong>优先级：</strong>活动规则 > 客户规则 > 全局规则 > 保底</p>
<p><strong>活动规则字段：</strong></p>
<ul>
  <li><strong>活动名称</strong>：活动的名字，方便识别</li>
  <li><strong>活动开始时间</strong>：生效开始时间</li>
  <li><strong>活动结束时间</strong>：失效时间</li>
</ul>
<p><strong>判断是否生效：</strong></p>
<p>计算时的系统时间在活动开始和结束之间 → 活动规则生效</p>
<p><strong>使用场景：</strong></p>
<ul>
  <li>双11、618等大促期间的优惠价</li>
  <li>新客户首月优惠</li>
  <li>季节性调价</li>
</ul>
<p>同一个客户可以有多条活动规则，按有效期自动判断。</p>`,
    related: ['运费计算完整流程', '规则管理'],
  },
  {
    keywords: ['启用', '禁用', '开关', '状态', 'is_enabled', '启用规则', '禁用规则'],
    title: '规则启用/禁用',
    answer: `<p><strong>启用/禁用</strong>控制规则是否参与运费计算。</p>
<p><strong>操作方式：</strong></p>
<ul>
  <li>阶段报价视图：每条规则右边有开关</li>
  <li>列表视图：表格里有启用/禁用开关列</li>
  <li>拉均重规则也有独立的开关</li>
</ul>
<p><strong>禁用后：</strong></p>
<ul>
  <li>规则数据保留，只是不参与计算</li>
  <li>列表里仍然能看到（灰色显示）</li>
  <li>可以随时重新启用</li>
  <li>省份不会消失</li>
</ul>
<p><strong>注意：</strong></p>
<ul>
  <li>禁用的规则在计算时会被跳过，继续找下一级匹配的规则</li>
  <li>如果该客户所有规则都禁用了，就会降级到全局规则</li>
</ul>`,
    related: ['删除规则', '规则管理', '运费计算完整流程'],
  },
  {
    keywords: ['删除', '删除规则', '删除客户', '批量删除', '删除确认'],
    title: '删除规则与客户',
    answer: `<p><strong>删除单条规则：</strong></p>
<ul>
  <li>点规则旁边的删除按钮</li>
  <li>会同时删除该规则的阶梯区间数据</li>
  <li>不可恢复，谨慎操作</li>
</ul>
<p><strong>批量删除：</strong></p>
<ul>
  <li>在列表视图勾选多条规则</li>
  <li>点「批量删除」按钮</li>
  <li>确认后一次性删除选中的所有规则</li>
</ul>
<p><strong>删除客户：</strong></p>
<ul>
  <li>在客户列表点删除按钮</li>
  <li>会删除该客户的所有规则（客户规则 + 活动规则）</li>
  <li>客户从列表中消失</li>
</ul>
<p><strong>不能删除的：</strong></p>
<ul>
  <li>系统默认保底规则（rule_type='default'）</li>
</ul>
<p><strong>建议：</strong>重要数据删除前先导出备份</p>`,
    related: ['规则导入导出', '启用禁用', '客户管理'],
  },
  {
    keywords: ['导入', '导出', '模板', 'excel', '批量', '批量导入'],
    title: '规则导入导出',
    answer: `<p><strong>导入模板格式（14列）：</strong></p>
<ol>
  <li>客户名称</li>
  <li>省份（空=全国）</li>
  <li>计费模式（simple/bracket）</li>
  <li>续重模式（full_kg/actual_weight/hundred_gram）</li>
  <li>首重(kg)</li>
  <li>首重单价(元)</li>
  <li>续重单价(元)</li>
  <li>保底价(元)</li>
  <li>最高价(元)</li>
  <li>附加费(元)</li>
  <li>区域名称</li>
  <li>规则类型（customer/campaign）</li>
  <li>启用（1/0）</li>
  <li>备注</li>
</ol>
<p><strong>bracket模式额外列（第15-22列）：</strong></p>
<ul>
  <li>0-0.5kg价、0.5-1kg价、1-2kg价、2-3kg价</li>
  <li>3-30kg首重价、3-30kg续重价</li>
  <li>30kg以上首重价、30kg以上续重价</li>
</ul>
<p><strong>导出：</strong>一键导出所有规则，含22列完整数据</p>
<p><strong>提示：</strong>点「下载模板」获取带示例的模板文件</p>`,
    related: ['区域模板生成', '复制规则', '客户管理'],
  },
  {
    keywords: ['区域模板生成', '生成区域', '批量生成', '模板生成'],
    title: '区域模板生成',
    answer: `<p><strong>区域模板生成</strong>是快速为客户创建所有省份规则的功能。</p>
<p><strong>操作步骤：</strong></p>
<ol>
  <li>选择客户</li>
  <li>设置6个区的价格方案（首重、续重、阶梯价等）</li>
  <li>点「生成规则」</li>
  <li>系统自动为每个区的每个省份创建一条规则</li>
</ol>
<p><strong>价格方案包含：</strong></p>
<ul>
  <li>首重/续重价格</li>
  <li>0-0.5kg、0.5-1kg、1-2kg、2-3kg 一口价</li>
  <li>3-30kg 首续重价</li>
  <li>30kg以上 首续重价</li>
  <li>保底价、附加费</li>
</ul>
<p><strong>注意事项：</strong></p>
<ul>
  <li>生成前会删除该客户已有的「区域型规则」（有zone_id的）</li>
  <li>非区域型规则（全国通配、自定义省份）不会被删</li>
  <li>生成的规则默认都是启用状态</li>
  <li>可以反复生成，每次覆盖之前的区域型规则</li>
</ul>`,
    related: ['区域规则', '阶梯计费模式', '客户管理'],
  },
  {
    keywords: ['规则测试', '测试', '试算', 'test', '快速测试'],
    title: '规则测试功能',
    answer: `<p><strong>规则测试</strong>不用导入Excel，直接输入参数就能试算运费，方便验证规则配置是否正确。</p>
<p><strong>单条测试：</strong></p>
<ul>
  <li>输入客户名、省份、重量</li>
  <li>显示最终运费、原始运费、加价金额</li>
  <li>显示命中的是哪条规则、什么优先级</li>
  <li>bracket模式显示命中的区间详情</li>
  <li>显示省份加价、全局加价等明细</li>
</ul>
<p><strong>批量测试：</strong></p>
<ul>
  <li>输入多个重量值（逗号分隔）</li>
  <li>选一个省份或测全部省份</li>
  <li>一次算出所有结果，方便对比价格走势</li>
  <li>可以直观看到阶梯定价的拐点是否正确</li>
</ul>
<p><strong>用途：</strong></p>
<ul>
  <li>配置完规则后验证价格是否正确</li>
  <li>排查某个包裹运费异常的原因</li>
  <li>对比不同客户/区域的价格差异</li>
</ul>`,
    related: ['运费计算完整流程', '阶梯计费模式'],
  },
  {
    keywords: ['计费结算', '批量计算', 'Excel导入', '计算', '结算', 'calc'],
    title: '计费结算（批量计算）',
    answer: `<p><strong>计费结算</strong>是导入Excel批量计算运费的功能。</p>
<p><strong>操作流程：</strong></p>
<ol>
  <li>上传Excel文件</li>
  <li>系统自动识别列（客户名、省份、重量等）</li>
  <li>点击开始计算</li>
  <li>等待计算完成（后台计算，可切换页面）</li>
  <li>查看结果，导出Excel</li>
</ol>
<p><strong>计算结果包含：</strong></p>
<ul>
  <li>原始数据 + 计算出的运费</li>
  <li>命中的规则信息</li>
  <li>加价明细</li>
</ul>
<p><strong>特点：</strong></p>
<ul>
  <li>计算在后台进行，不阻塞操作</li>
  <li>顶部有计算中横幅提示</li>
  <li>计算历史会自动保存，可随时回看</li>
  <li>支持大文件批量处理</li>
</ul>`,
    related: ['计算历史', '规则测试', '拉均重'],
  },
  {
    keywords: ['计算历史', '历史记录', 'history', '历史'],
    title: '计算历史',
    answer: `<p><strong>计算历史</strong>保存每次批量计算的记录，可以随时回看和导出。</p>
<p><strong>历史记录包含：</strong></p>
<ul>
  <li>任务名称/文件名</li>
  <li>计算时间</li>
  <li>总条数</li>
  <li>计算状态（完成/进行中/失败）</li>
</ul>
<p><strong>可以做的操作：</strong></p>
<ul>
  <li>查看详情：看到计算结果的完整列表</li>
  <li>导出Excel：把计算结果下载下来</li>
  <li>删除记录：清理不需要的历史</li>
</ul>
<p><strong>数据存储：</strong>存在本地数据库的 calc_history 表</p>`,
    related: ['计费结算', '数据存储说明'],
  },
  {
    keywords: ['数据库', '数据存哪', 'yunfei.db', 'sqlite', '数据目录', '数据文件'],
    title: '数据存储说明',
    answer: `<p><strong>数据库类型：</strong>SQLite（单文件数据库，绿色免安装）</p>
<p><strong>数据文件位置：</strong></p>
<ul>
  <li>Windows：<code>C:\\Users\\用户名\\AppData\\Roaming\\yunfei\\yunfei.db</code></li>
  <li>Mac：<code>~/yunfei/yunfei.db</code></li>
</ul>
<p><strong>数据库主要表：</strong></p>
<ul>
  <li>freight_rules - 运费规则主表</li>
  <li>freight_weight_brackets - 阶梯区间价格表</li>
  <li>freight_zones - 区域表</li>
  <li>freight_zone_provinces - 区域省份映射表</li>
  <li>global_rules - 全局规则（保底+加价）</li>
  <li>global_province_surcharges - 全局省份加价</li>
  <li>avg_weight_rules - 拉均重规则</li>
  <li>calc_history - 计算历史</li>
  <li>license_info - 授权信息</li>
  <li>app_settings - 系统设置</li>
</ul>
<p><strong>备份：</strong>直接复制 yunfei.db 文件就是完整备份</p>
<p><strong>删除/重置：</strong>删掉 yunfei.db，下次启动自动重建（所有配置丢失）</p>
<p><strong>相关文件：</strong></p>
<ul>
  <li>yunfei.db-wal - 写入日志文件（WAL模式）</li>
  <li>yunfei.db-shm - 共享内存文件</li>
  <li>settings.json - 部分设置</li>
  <li>license.dat - 授权文件</li>
</ul>`,
    related: ['系统设置', '授权与激活'],
  },
  {
    keywords: ['系统设置', '设置', 'settings', '全局规则配置', '省份加价配置'],
    title: '系统设置',
    answer: `<p><strong>系统设置</strong>页面管理全局配置。</p>
<p><strong>全局规则配置：</strong></p>
<ul>
  <li>默认首重/续重价格（保底用）</li>
  <li>默认保底价</li>
  <li>零重价格（重量为0时的价格）</li>
  <li>全局固定加价</li>
  <li>全局百分比加价</li>
</ul>
<p><strong>省份加价配置：</strong></p>
<ul>
  <li>添加/编辑/删除省份加价</li>
  <li>每个省份独立设置金额</li>
</ul>
<p><strong>其他功能：</strong></p>
<ul>
  <li>清空计算历史</li>
  <li>数据库信息查看</li>
</ul>`,
    related: ['全局规则', '省份加价', '全局加价'],
  },
  {
    keywords: ['授权', 'license', '激活', '机器码', '授权码', '注册', '怎么授权', '怎么激活', '怎么注册'],
    title: '软件授权与激活',
    answer: `<p><strong>授权方式：</strong>基于机器码的授权认证，一机一码</p>
<p><strong>授权状态说明：</strong></p>
<ul>
  <li>🟢 <strong>正常使用</strong> - 绿色状态点，授权有效</li>
  <li>🟡 <strong>即将过期</strong> - 黄色闪烁，剩余7天内</li>
  <li>🔴 <strong>已过期</strong> - 红色，无法使用</li>
  <li>⚪ <strong>未激活</strong> - 需要导入授权码</li>
</ul>
<p><strong>注册激活步骤：</strong></p>
<ol>
  <li>打开软件，点击左侧菜单的「授权管理」</li>
  <li>复制页面上显示的<strong>机器码</strong>（每台电脑唯一）</li>
  <li>联系客服 17771300068，提供机器码和客户名称</li>
  <li>客服生成授权码后发给你</li>
  <li>把授权码粘贴到输入框，点击「激活」</li>
  <li>提示激活成功就可以正常使用了</li>
</ol>
<p><strong>常见问题：</strong></p>
<ul>
  <li><strong>换电脑了怎么办？</strong>联系客服重新授权，需要提供新机器码</li>
  <li><strong>断网能用吗？</strong>可以，在线验证后缓存7天，期间断网不影响</li>
  <li><strong>授权过期了怎么办？</strong>联系客服续费</li>
  <li><strong>重装系统后还能用吗？</strong>硬件不变的话机器码不变，可以重新激活</li>
</ul>
<p><strong>授权文件位置：</strong>和数据库在同一个目录（yunfei文件夹）下的 license.dat</p>`,
    related: ['数据存储说明', '系统设置'],
  },
]


// ============ 对话逻辑 ============
function initChat() {
  messages.value = [{
    role: 'ai',
    content: `<p>👋 你好呀！我是喵喵知识库小助手 🐵</p>
<p>有什么关于运费规则的问题都可以问我~</p>
<p>试试下面的快捷问题吧 👇</p>`,
    suggestions: quickQuestions,
  }]
}

function findAnswer(question: string): KBItem | null {
  const q = question.toLowerCase().trim()
  let bestMatch: KBItem | null = null
  let bestScore = 0

  for (const item of knowledgeBase) {
    let score = 0
    for (const kw of item.keywords) {
      if (q.includes(kw.toLowerCase())) {
        score += kw.length
      }
    }
    if (item.title.toLowerCase().includes(q)) {
      score += 20
    }
    if (score > bestScore) {
      bestScore = score
      bestMatch = item
    }
  }

  return bestScore >= 2 ? bestMatch : null
}

function generateAnswer(question: string): { content: string; suggestions?: string[] } {
  const answer = findAnswer(question)
  if (answer) {
    let content = `<p><strong>📌 ${answer.title}</strong></p>${answer.answer}`
    const suggestions = answer.related || []
    return { content, suggestions }
  }
  return {
    content: `<p>😅 这个问题我还不太确定呢...</p>
<p>你可以问问我这些方面的问题：</p>
<ul>
  <li>运费怎么算的 / 计费规则</li>
  <li>阶梯定价 / 区间计费</li>
  <li>拉均重 / 偏差加价</li>
  <li>区域规则 / 六区体系</li>
  <li>全局规则 / 优先级</li>
  <li>导入导出 / 模板</li>
  <li>规则测试</li>
  <li>数据库 / 数据存储</li>
</ul>
<p>或者换个问法试试~</p>`,
    suggestions: quickQuestions,
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (chatBodyRef.value) {
      chatBodyRef.value.scrollTop = chatBodyRef.value.scrollHeight
    }
  })
}

function sendMsg(text: string) {
  if (!text.trim()) return
  messages.value.push({ role: 'user', content: text })
  inputText.value = ''
  scrollToBottom()

  thinking.value = true
  setTimeout(() => {
    const result = generateAnswer(text)
    messages.value.push({
      role: 'ai',
      content: result.content,
      suggestions: result.suggestions,
    })
    thinking.value = false
    scrollToBottom()
  }, 300 + Math.random() * 400)
}

function handleSend() {
  if (!inputText.value.trim()) return
  sendMsg(inputText.value)
}

function clearChat() {
  initChat()
  scrollToBottom()
}

function handleIconClick() {
  if (movedDuringDrag) return
  panelOpen.value = !panelOpen.value
  if (panelOpen.value) {
    showBubble.value = false
    scrollToBottom()
  }
}

function closePanel() {
  panelOpen.value = false
}

// ============ 拖动逻辑 ============
function startDrag(e: MouseEvent | TouchEvent) {
  dragging = true
  dragTarget = 'icon'
  movedDuringDrag = false
  const p = getEventPos(e)
  startX = p.x
  startY = p.y
  startPosX = pos.x
  startPosY = pos.y

  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
  document.addEventListener('touchmove', onDrag, { passive: false })
  document.addEventListener('touchend', stopDrag)
}

function startDragPanel(e: MouseEvent | TouchEvent) {
  e.preventDefault()
  dragging = true
  dragTarget = 'panel'
  movedDuringDrag = false
  const p = getEventPos(e)
  startX = p.x
  startY = p.y
  startPosX = panelPos.x
  startPosY = panelPos.y

  document.addEventListener('mousemove', onDrag)
  document.addEventListener('mouseup', stopDrag)
  document.addEventListener('touchmove', onDrag, { passive: false })
  document.addEventListener('touchend', stopDrag)
}

function getEventPos(e: MouseEvent | TouchEvent) {
  if ('touches' in e && e.touches.length > 0) {
    return { x: e.touches[0].clientX, y: e.touches[0].clientY }
  }
  return { x: (e as MouseEvent).clientX, y: (e as MouseEvent).clientY }
}

function onDrag(e: MouseEvent | TouchEvent) {
  if (!dragging) return
  e.preventDefault()
  const p = getEventPos(e)
  const dx = p.x - startX
  const dy = p.y - startY

  if (Math.abs(dx) > 3 || Math.abs(dy) > 3) {
    movedDuringDrag = true
  }

  if (dragTarget === 'icon') {
    pos.x = Math.max(0, Math.min(window.innerWidth - 60, startPosX + dx))
    pos.y = Math.max(0, Math.min(window.innerHeight - 60, startPosY + dy))
  } else {
    panelPos.x = Math.max(0, Math.min(window.innerWidth - 380, startPosX + dx))
    panelPos.y = Math.max(0, Math.min(window.innerHeight - 300, startPosY + dy))
  }
}

function stopDrag() {
  dragging = false
  document.removeEventListener('mousemove', onDrag)
  document.removeEventListener('mouseup', stopDrag)
  document.removeEventListener('touchmove', onDrag)
  document.removeEventListener('touchend', stopDrag)
}

onMounted(() => {
  initChat()
  setTimeout(() => {
    showBubble.value = false
  }, 5000)
})
</script>

<style scoped>
.kai-ai-wrap {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  pointer-events: none;
  z-index: 9999;
}

/* 浮动猴子图标 */
.monkey-float {
  position: fixed;
  width: 56px;
  height: 56px;
  cursor: pointer;
  pointer-events: auto;
  z-index: 9999;
  user-select: none;
  transition: transform 0.15s;
}
.monkey-float:hover {
  transform: scale(1.1);
}
.monkey-float:active {
  transform: scale(0.95);
}
.monkey-icon {
  width: 56px;
  height: 56px;
  border-radius: 50%;
  background: linear-gradient(135deg, #ffd966, #ffb347);
  box-shadow: 0 4px 16px rgba(255, 140, 0, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  padding: 4px;
  box-sizing: border-box;
  border: 3px solid #fff;
}
.monkey-icon img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.monkey-bubble {
  position: absolute;
  left: 60px;
  top: 50%;
  transform: translateY(-50%);
  background: #fff;
  padding: 6px 12px;
  border-radius: 16px;
  font-size: 12px;
  color: #606266;
  white-space: nowrap;
  box-shadow: 0 2px 12px rgba(0,0,0,0.1);
  animation: bubbleFloat 2s ease-in-out infinite;
}
.monkey-bubble::before {
  content: '';
  position: absolute;
  left: -6px;
  top: 50%;
  transform: translateY(-50%);
  border: 6px solid transparent;
  border-right-color: #fff;
  border-left: none;
}
@keyframes bubbleFloat {
  0%, 100% { transform: translateY(-50%); }
  50% { transform: translateY(-58%); }
}

/* 问答面板 */
.ai-panel {
  position: fixed;
  width: 360px;
  height: 520px;
  background: #fff;
  border-radius: 14px;
  box-shadow: 0 8px 32px rgba(0,0,0,0.18);
  display: flex;
  flex-direction: column;
  pointer-events: auto;
  z-index: 9998;
  overflow: hidden;
}
.ai-panel-header {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
  padding: 14px 16px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  cursor: move;
  user-select: none;
}
.ai-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 600;
  font-size: 15px;
}
.ai-title-icon {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: rgba(255,255,255,0.25);
  padding: 2px;
}
.ai-actions {
  display: flex;
  gap: 4px;
}
.ai-action-btn {
  width: 28px;
  height: 28px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
  transition: background 0.2s;
}
.ai-action-btn:hover {
  background: rgba(255,255,255,0.2);
}
.ai-action-btn.close-btn:hover {
  background: rgba(255, 100, 100, 0.3);
}

.ai-panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 14px;
  background: #f7f8fa;
}
.msg-wrap {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.msg-item {
  display: flex;
  gap: 8px;
  max-width: 85%;
}
.msg-item.user {
  align-self: flex-end;
  flex-direction: row-reverse;
}
.msg-avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  background: linear-gradient(135deg, #ffd966, #ffb347);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  box-sizing: border-box;
}
.msg-avatar img {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.msg-avatar.user {
  background: #409eff;
  color: #fff;
  padding: 0;
}
.msg-bubble {
  background: #fff;
  padding: 10px 12px;
  border-radius: 10px;
  font-size: 13px;
  line-height: 1.6;
  color: #303133;
  box-shadow: 0 1px 4px rgba(0,0,0,0.06);
  word-break: break-word;
}
.msg-item.user .msg-bubble {
  background: linear-gradient(135deg, #667eea, #764ba2);
  color: #fff;
}
.msg-bubble p {
  margin: 4px 0;
}
.msg-bubble ul, .msg-bubble ol {
  margin: 4px 0;
  padding-left: 18px;
}
.msg-bubble li {
  margin: 2px 0;
}
.msg-bubble strong {
  font-weight: 600;
}
.msg-bubble pre {
  background: rgba(0,0,0,0.05);
  padding: 8px 10px;
  border-radius: 6px;
  font-size: 12px;
  overflow-x: auto;
  margin: 6px 0;
}
.msg-item.user .msg-bubble pre {
  background: rgba(255,255,255,0.15);
}
.msg-bubble code {
  background: rgba(0,0,0,0.06);
  padding: 1px 4px;
  border-radius: 3px;
  font-size: 12px;
}
.msg-item.user .msg-bubble code {
  background: rgba(255,255,255,0.2);
}
.msg-suggestions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 8px;
  padding-top: 8px;
  border-top: 1px solid rgba(0,0,0,0.06);
}
.sug-item {
  padding: 4px 10px;
  background: #ecf5ff;
  color: #409eff;
  border-radius: 12px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.sug-item:hover {
  background: #409eff;
  color: #fff;
}

.msg-bubble.thinking {
  display: flex;
  gap: 4px;
  padding: 14px 16px;
}
.msg-bubble.thinking .dot {
  width: 6px;
  height: 6px;
  background: #909399;
  border-radius: 50%;
  animation: thinkBounce 1.2s infinite;
}
.msg-bubble.thinking .dot:nth-child(2) { animation-delay: 0.15s; }
.msg-bubble.thinking .dot:nth-child(3) { animation-delay: 0.3s; }
@keyframes thinkBounce {
  0%, 80%, 100% { transform: translateY(0); opacity: 0.4; }
  40% { transform: translateY(-4px); opacity: 1; }
}

.ai-panel-footer {
  border-top: 1px solid #ebeef5;
  padding: 10px 12px;
  background: #fff;
}
.quick-questions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-bottom: 10px;
}
.quick-q {
  padding: 3px 10px;
  background: #f0f2f5;
  color: #606266;
  border-radius: 12px;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.quick-q:hover {
  background: #e0e4ea;
  color: #303133;
}
.input-row :deep(.el-input-group__append) {
  padding: 0 8px;
}

/* 面板动画 */
.ai-panel-enter-active,
.ai-panel-leave-active {
  transition: opacity 0.2s, transform 0.2s;
}
.ai-panel-enter-from,
.ai-panel-leave-to {
  opacity: 0;
  transform: scale(0.9);
}
</style>
