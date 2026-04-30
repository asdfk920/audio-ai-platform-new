<template>
  <BasicLayout>
    <template #wrapper>
      <el-card class="box-card">
        <div class="toolbar">
          <div class="title">会员管理</div>
          <div class="actions">
            <el-button size="mini" icon="el-icon-refresh" @click="reloadAll">刷新</el-button>
          </div>
        </div>

        <el-tabs v-model="active">
          <el-tab-pane label="会员等级" name="levels">
            <el-alert
              class="mb12"
              type="info"
              :closable="false"
              show-icon
              title="说明"
              description="列表含用户数、权益数等统计；价格/默认有效期等为运营参考字段（实际扣费与到期以订单为准）。请先执行数据库迁移 039（member_level 扩展字段）。"
            />
            <el-form :inline="true" class="benefit-toolbar mb12">
              <el-form-item label="关键词">
                <el-input
                  v-model="levelQuery.keyword"
                  size="small"
                  clearable
                  placeholder="编码 / 名称 / 价格文案"
                  style="width: 200px"
                  @keyup.enter.native="noop"
                />
              </el-form-item>
              <el-form-item label="状态">
                <el-select v-model="levelQuery.status" size="small" clearable placeholder="全部" style="width: 110px">
                  <el-option label="启用" :value="1" />
                  <el-option label="禁用" :value="0" />
                </el-select>
              </el-form-item>
              <el-form-item label="获取方式">
                <el-select v-model="levelQuery.acquireType" size="small" clearable placeholder="全部" style="width: 130px">
                  <el-option label="注册即有" :value="1" />
                  <el-option label="付费购买" :value="2" />
                  <el-option label="活动赠送" :value="3" />
                  <el-option label="邀请解锁" :value="4" />
                </el-select>
              </el-form-item>
              <el-form-item label="列表排序">
                <el-select v-model="levelSortBy" size="small" style="width: 140px">
                  <el-option label="后台排序号" value="sort" />
                  <el-option label="用户数" value="user_count" />
                  <el-option label="权益数" value="benefit_count" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" size="mini" icon="el-icon-plus" @click="openLevelDialog()">新增等级</el-button>
                <el-button size="mini" :disabled="!levelSelection.length" @click="batchLevelStatus(1)">批量启用</el-button>
                <el-button size="mini" :disabled="!levelSelection.length" @click="batchLevelStatus(0)">批量禁用</el-button>
                <el-button size="mini" :disabled="levelSelection.length !== 1" @click="copyLevelFromSelection">复制选中</el-button>
                <el-button size="mini" @click="exportLevelsCsv">导出 CSV</el-button>
              </el-form-item>
            </el-form>
            <el-table
              v-loading="loading.levels"
              :data="filteredLevels"
              border
              @selection-change="onLevelSelectionChange"
            >
              <el-table-column type="selection" width="48" fixed />
              <el-table-column label="编码" prop="level_code" min-width="120" show-overflow-tooltip />
              <el-table-column label="名称" prop="level_name" min-width="110" show-overflow-tooltip />
              <el-table-column label="排序" prop="sort" width="72" />
              <el-table-column label="版本" prop="version" width="64" />
              <el-table-column label="状态" width="88">
                <template slot-scope="{ row }">
                  <el-tag v-if="row.status === 1" type="success" size="mini">启用</el-tag>
                  <el-tag v-else type="info" size="mini">禁用</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="获取方式" min-width="100">
                <template slot-scope="{ row }">
                  <el-tag size="mini" type="info">{{ acquireTypeLabel(row.acquire_type) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="价格/套餐" min-width="140" show-overflow-tooltip>
                <template slot-scope="{ row }">
                  {{ row.price_package_hint || '—' }}
                </template>
              </el-table-column>
              <el-table-column label="默认有效期" width="110">
                <template slot-scope="{ row }">
                  {{ formatDefaultValidityDays(row.default_validity_days) }}
                </template>
              </el-table-column>
              <el-table-column label="用户数" width="80">
                <template slot-scope="{ row }">
                  {{ row.user_count != null ? row.user_count : 0 }}
                </template>
              </el-table-column>
              <el-table-column label="权益数" width="80">
                <template slot-scope="{ row }">
                  {{ row.benefit_count != null ? row.benefit_count : 0 }}
                </template>
              </el-table-column>
              <el-table-column label="调整排序" width="120" fixed="right">
                <template slot-scope="{ row }">
                  <el-button size="mini" type="text" :disabled="!canMoveLevel(row, -1)" @click="moveLevelSort(row, -1)">上移</el-button>
                  <el-button size="mini" type="text" :disabled="!canMoveLevel(row, 1)" @click="moveLevelSort(row, 1)">下移</el-button>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="260" fixed="right">
                <template slot-scope="{ row }">
                  <el-button size="mini" type="text" icon="el-icon-edit" @click="openLevelDialog(row)">编辑</el-button>
                  <el-button size="mini" type="text" icon="el-icon-link" @click="goToLevelMapping(row)">权益配置</el-button>
                  <el-button size="mini" type="text" icon="el-icon-delete" @click="deleteLevel(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
            <el-collapse class="mapping-meta">
              <el-collapse-item title="等级操作日志（最近 20 条，仅本浏览器）" name="levlog">
                <el-table :data="levelOpLog" size="mini" border max-height="200">
                  <el-table-column prop="time" label="时间" width="160" />
                  <el-table-column prop="detail" label="操作" min-width="280" />
                </el-table>
              </el-collapse-item>
            </el-collapse>
          </el-tab-pane>

          <el-tab-pane label="会员权益" name="benefits">
            <el-alert
              class="mb12"
              type="info"
              :closable="false"
              show-icon
              title="说明"
              description="生命周期为「已上线」且在生效时间窗内的权益，才会对用户端展示。互斥/依赖等规则写在扩展 JSON 中。请先执行数据库迁移 038（member_benefit 扩展字段）。"
            />
            <el-form :inline="true" class="benefit-toolbar mb12">
              <el-form-item label="关键词">
                <el-input
                  v-model="benefitQuery.keyword"
                  size="small"
                  clearable
                  placeholder="编码 / 名称 / 说明"
                  style="width: 220px"
                  @keyup.enter.native="noop"
                />
              </el-form-item>
              <el-form-item label="生命周期">
                <el-select v-model="benefitQuery.lifecycle" size="small" clearable placeholder="全部" style="width: 130px">
                  <el-option label="草稿" :value="0" />
                  <el-option label="待发布" :value="1" />
                  <el-option label="已上线" :value="2" />
                  <el-option label="已下线" :value="3" />
                </el-select>
              </el-form-item>
              <el-form-item label="类型">
                <el-select v-model="benefitQuery.benefitType" size="small" clearable placeholder="全部" style="width: 130px">
                  <el-option label="基础功能" :value="1" />
                  <el-option label="增值服务" :value="2" />
                  <el-option label="限时活动" :value="3" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-checkbox v-model="benefitShowMetrics">显示数据指标列（占位）</el-checkbox>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" size="mini" icon="el-icon-plus" @click="openBenefitDialog()">新增权益</el-button>
                <el-button size="mini" :disabled="!benefitSelection.length" @click="batchBenefitLifecycle(2)">批量上线</el-button>
                <el-button size="mini" :disabled="!benefitSelection.length" @click="batchBenefitLifecycle(3)">批量下线</el-button>
                <el-button size="mini" :disabled="!benefitSelection.length" @click="batchBenefitType(1)">标为基础</el-button>
                <el-button size="mini" :disabled="!benefitSelection.length" @click="batchBenefitType(2)">标为增值</el-button>
                <el-button size="mini" :disabled="!benefitSelection.length" @click="exportBenefitsCsv">导出 CSV</el-button>
                <el-button size="mini" :disabled="benefitSelection.length !== 1" @click="copyBenefitFromSelection">复制选中</el-button>
              </el-form-item>
            </el-form>
            <el-table
              v-loading="loading.benefits"
              :data="filteredBenefits"
              border
              @selection-change="onBenefitSelectionChange"
            >
              <el-table-column type="selection" width="48" fixed />
              <el-table-column label="编码" prop="benefit_code" min-width="150" show-overflow-tooltip />
              <el-table-column label="名称" prop="benefit_name" min-width="120" show-overflow-tooltip />
              <el-table-column label="类型" width="100">
                <template slot-scope="{ row }">
                  <el-tag size="mini" type="info">{{ benefitTypeLabel(row.benefit_type) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="生命周期" width="100">
                <template slot-scope="{ row }">
                  <el-tag size="mini" :type="lifecycleTagType(row.lifecycle_status)">{{ lifecycleLabel(row.lifecycle_status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="版本" prop="version" width="70" />
              <el-table-column label="标签" prop="tags" min-width="120" show-overflow-tooltip />
              <el-table-column label="生效时间" min-width="180" show-overflow-tooltip>
                <template slot-scope="{ row }">
                  {{ formatEffectRange(row) }}
                </template>
              </el-table-column>
              <el-table-column label="说明" prop="description" min-width="160" show-overflow-tooltip />
              <el-table-column v-if="benefitShowMetrics" label="使用率" width="88">
                <template slot-scope>
                  <span title="P2：接入埋点后展示">—</span>
                </template>
              </el-table-column>
              <el-table-column v-if="benefitShowMetrics" label="满意度" width="88">
                <template slot-scope>
                  <span title="P2">—</span>
                </template>
              </el-table-column>
              <el-table-column v-if="benefitShowMetrics" label="投诉率" width="88">
                <template slot-scope>
                  <span title="P2">—</span>
                </template>
              </el-table-column>
              <el-table-column v-if="benefitShowMetrics" label="付费转化" width="88">
                <template slot-scope>
                  <span title="P2">—</span>
                </template>
              </el-table-column>
              <el-table-column label="操作" width="240" fixed="right">
                <template slot-scope="{ row }">
                  <el-button size="mini" type="text" icon="el-icon-edit" @click="openBenefitDialog(row)">编辑</el-button>
                  <el-button size="mini" type="text" @click="confirmToggleBenefitOffline(row)">{{ (row.lifecycle_status === 2) ? '下线' : '上线' }}</el-button>
                  <el-button size="mini" type="text" icon="el-icon-delete" @click="deleteBenefit(row)">删除</el-button>
                </template>
              </el-table-column>
            </el-table>
            <el-collapse class="mapping-meta">
              <el-collapse-item title="权益操作日志（最近 20 条，仅本浏览器）" name="blog">
                <el-table :data="benefitOpLog" size="mini" border max-height="200">
                  <el-table-column prop="time" label="时间" width="160" />
                  <el-table-column prop="detail" label="操作" min-width="280" />
                </el-table>
              </el-collapse-item>
            </el-collapse>
          </el-tab-pane>

          <el-tab-pane label="等级权益配置" name="mapping">
            <el-alert
              class="mb12"
              type="info"
              :closable="false"
              show-icon
              title="操作指南"
              description="选择等级后，为该等级勾选可用权益；高等级默认继承低等级权益（见下方「已继承」区），可在此基础上勾选专属权益；保存前支持预览/对比。本地操作记录仅保存在当前浏览器，服务端审计为后续能力。"
            />

            <div class="mapping-toolbar">
              <div class="mapping-toolbar__left">
                <span class="mapping-label">等级</span>
                <el-select
                  v-model="mapping.level_code"
                  placeholder="请选择等级"
                  size="small"
                  filterable
                  class="mapping-level-select"
                  @change="onMappingLevelChange"
                >
                  <el-option
                    v-for="l in sortedLevelsForMapping"
                    :key="l.level_code"
                    :label="levelOptionLabel(l)"
                    :value="l.level_code"
                  >
                    <div class="level-option">
                      <div class="level-option__title">{{ l.level_name }}（{{ l.level_code }}）</div>
                      <div class="level-option__sub">排序值 {{ l.sort }} · {{ levelValidityHint(l) }}</div>
                    </div>
                  </el-option>
                </el-select>
                <span v-if="mapping.level_code && currentLevelRow" class="mapping-current-hint">
                  当前：{{ currentLevelRow.level_name }} · 编码 {{ currentLevelRow.level_code }} · 排序 {{ currentLevelRow.sort }}
                </span>
              </div>
              <div class="mapping-toolbar__actions">
                <el-button size="mini" :disabled="!mapping.level_code || !mappingSelectableExclusiveBenefits.length" @click="selectAllMappingBenefits(true)">全选</el-button>
                <el-button size="mini" :disabled="!mapping.level_code || !mappingSelectableExclusiveBenefits.length" @click="invertMappingBenefits">反选</el-button>
                <el-button size="mini" :disabled="!mapping.level_code" @click="openPreviewDialog">配置预览</el-button>
                <el-button size="mini" :disabled="!mapping.level_code" @click="openCompareDialog">配置对比</el-button>
                <el-button size="mini" :disabled="!canUndoMapping" @click="undoLastMappingSave">撤销上次保存</el-button>
                <el-button type="primary" size="mini" :disabled="!mapping.level_code" @click="confirmSaveMapping">保存配置</el-button>
              </div>
            </div>

            <el-alert
              v-for="(w, idx) in mappingConflictWarnings"
              :key="'cw-' + idx"
              class="mb12"
              type="warning"
              :closable="true"
              show-icon
              :title="w"
            />

            <div v-loading="loading.mapping" class="benefit-mapping-body">
              <div class="mapping-section">
                <div class="mapping-section__title">
                  <i class="el-icon-lock mapping-section__icon" />
                  已继承权益（低等级权益，默认包含，不可取消）
                </div>
                <p v-if="mappingInheritedFromLevels && mappingInheritedFromLevels.length" class="mapping-inherit-hint">
                  继承自低等级：
                  <span v-for="(lv, i) in mappingInheritedFromLevels" :key="lv.level_code">
                    {{ lv.level_name }}（{{ lv.level_code }}）<span v-if="i < mappingInheritedFromLevels.length - 1">、</span>
                  </span>
                </p>
                <p v-else class="mapping-inherit-hint muted">当前为最低档或尚无低等级配置，无强制继承项。</p>
                <template v-for="group in inheritedGroupedForMapping">
                  <div v-if="group.items.length" :key="'inh-'+group.key" class="benefit-group">
                    <div class="benefit-group__title">
                      <i :class="group.icon" />
                      {{ group.label }}
                    </div>
                    <div class="benefit-grid">
                      <div
                        v-for="b in group.items"
                        :key="'inh-'+b.benefit_code"
                        class="benefit-card is-inherited"
                        :class="{ 'is-offline': !mappingBenefitSelectable(b) }"
                      >
                        <div class="benefit-card__inner">
                          <div class="benefit-inherited-head">
                            <i class="el-icon-lock benefit-lock" />
                            <span class="benefit-card__icon"><i :class="benefitIcon(b.benefit_code)" /></span>
                            <span class="benefit-card__name">{{ b.benefit_name }}</span>
                          </div>
                          <div class="benefit-card__code">{{ b.benefit_code }}</div>
                          <div class="benefit-card__desc">{{ b.description || '暂无说明' }}</div>
                          <el-tag size="mini" type="info" effect="plain">自动继承 · 不可取消</el-tag>
                        </div>
                      </div>
                    </div>
                  </div>
                </template>
              </div>

              <div class="mapping-section mapping-section--exclusive">
                <div class="mapping-section__title">
                  <i class="el-icon-s-check mapping-section__icon" />
                  当前等级专属权益（可勾选）
                </div>
                <el-checkbox-group v-model="exclusiveBenefitCodes" class="benefit-checkbox-group">
                  <template v-for="group in exclusiveGroupedForMapping">
                    <div v-if="group.items.length" :key="'exc-'+group.key" class="benefit-group">
                      <div class="benefit-group__title">
                        <i :class="group.icon" />
                        {{ group.label }}
                        <span class="benefit-group__hint">{{ group.hint }}</span>
                      </div>
                      <div class="benefit-grid">
                        <div
                          v-for="b in group.items"
                          :key="b.benefit_code"
                          class="benefit-card"
                          :class="{
                            'is-offline': !mappingBenefitSelectable(b),
                            'is-configured': isBenefitChecked(b.benefit_code)
                          }"
                        >
                          <el-tooltip placement="top" :disabled="!mappingBenefitSelectable(b)">
                            <div slot="content" class="benefit-tooltip">
                              <div class="benefit-tooltip__title">{{ b.benefit_name }}（{{ b.benefit_code }}）</div>
                              <p>{{ benefitDetailRule(b) }}</p>
                              <p v-if="b.description" class="benefit-tooltip__desc">{{ b.description }}</p>
                            </div>
                            <div class="benefit-card__inner">
                              <el-checkbox :label="b.benefit_code" :disabled="!mappingBenefitSelectable(b)">
                                <span class="benefit-card__icon"><i :class="benefitIcon(b.benefit_code)" /></span>
                                <span class="benefit-card__name">{{ b.benefit_name }}</span>
                              </el-checkbox>
                              <div class="benefit-card__code">{{ b.benefit_code }}</div>
                              <div class="benefit-card__desc">{{ b.description || '暂无说明，可在「会员权益」中编辑' }}</div>
                              <div class="benefit-card__meta">
                                <span v-for="(hint, hi) in benefitExtraHints(b)" :key="'h-'+b.benefit_code+'-'+hi" class="benefit-meta-line">{{ hint }}</span>
                              </div>
                              <div class="benefit-card__actions">
                                <el-button type="text" size="mini" @click="openBenefitRuleDrawer(b)">规则说明</el-button>
                                <el-button v-if="b.id" type="text" size="mini" @click="goEditBenefitFromMapping(b)">去编辑权益</el-button>
                              </div>
                              <el-tag v-if="!mappingBenefitSelectable(b)" size="mini" type="info">已下线 · 不可配置</el-tag>
                              <el-tag v-else-if="!isBenefitChecked(b.benefit_code)" size="mini" type="info" effect="plain">未勾选</el-tag>
                            </div>
                          </el-tooltip>
                        </div>
                      </div>
                    </div>
                  </template>
                </el-checkbox-group>
              </div>
            </div>

            <div class="mapping-footer-bar">
              <el-button type="primary" size="small" :disabled="!mapping.level_code" @click="confirmSaveMapping">保存配置</el-button>
              <el-button size="small" :disabled="!mapping.level_code" @click="cancelMappingEdits">取消</el-button>
              <el-button type="text" size="small" @click="openMappingLogPanel">查看全局操作日志</el-button>
            </div>

            <el-collapse v-model="mappingCollapseNames" class="mapping-meta">
              <el-collapse-item ref="mappingLogCollapse" title="本地操作记录（最近 20 条，仅本浏览器）" name="log">
                <el-table :data="mappingOpLog" size="mini" border max-height="220">
                  <el-table-column prop="time" label="时间" width="160" />
                  <el-table-column prop="level" label="等级" width="120" />
                  <el-table-column prop="detail" label="操作" min-width="200" />
                </el-table>
              </el-collapse-item>
            </el-collapse>
          </el-tab-pane>

          <el-tab-pane label="用户会员" name="user">
            <el-row :gutter="12" class="um-summary mb12">
              <el-col :span="4">
                <div class="um-stat"><span class="um-stat__label">总记录</span><span class="um-stat__val">{{ umSummary.total != null ? umSummary.total : '—' }}</span></div>
              </el-col>
              <el-col :span="4">
                <div class="um-stat"><span class="um-stat__label">生效中</span><span class="um-stat__val um-stat__ok">{{ umSummary.active != null ? umSummary.active : '—' }}</span></div>
              </el-col>
              <el-col :span="4">
                <div class="um-stat"><span class="um-stat__label">已过期</span><span class="um-stat__val">{{ umSummary.expired != null ? umSummary.expired : '—' }}</span></div>
              </el-col>
              <el-col :span="4">
                <div class="um-stat"><span class="um-stat__label">即将到期(7天)</span><span class="um-stat__val um-stat__warn">{{ umSummary.expiring_soon != null ? umSummary.expiring_soon : '—' }}</span></div>
              </el-col>
              <el-col :span="4">
                <div class="um-stat"><span class="um-stat__label">冻结</span><span class="um-stat__val">{{ umSummary.frozen != null ? umSummary.frozen : '—' }}</span></div>
              </el-col>
            </el-row>
            <el-form :inline="true" class="benefit-toolbar mb12">
              <el-form-item label="关键词">
                <el-input
                  v-model="umFilters.keyword"
                  size="small"
                  clearable
                  placeholder="用户ID / 昵称 / 手机号"
                  style="width: 200px"
                  @keyup.enter.native="umSearch"
                />
              </el-form-item>
              <el-form-item label="等级">
                <el-select v-model="umFilters.level_code" size="small" clearable placeholder="全部" style="width: 140px" filterable>
                  <el-option v-for="l in levels" :key="'um-'+l.level_code" :label="`${l.level_name}（${l.level_code}）`" :value="l.level_code" />
                </el-select>
              </el-form-item>
              <el-form-item label="有效期状态">
                <el-select v-model="umFilters.validity_status" size="small" clearable placeholder="全部" style="width: 120px">
                  <el-option label="生效中" value="active" />
                  <el-option label="已过期" value="expired" />
                  <el-option label="即将到期" value="expiring_soon" />
                  <el-option label="冻结" value="frozen" />
                </el-select>
              </el-form-item>
              <el-form-item label="发放渠道">
                <el-select v-model="umFilters.register_type" size="small" clearable placeholder="全部" style="width: 120px">
                  <el-option label="自购 pay" value="pay" />
                  <el-option label="活动 gift" value="gift" />
                  <el-option label="人工 admin" value="admin" />
                  <el-option label="注册 register" value="register" />
                </el-select>
              </el-form-item>
              <el-form-item label="开通时间">
                <el-date-picker
                  v-model="umFilters.createdRange"
                  type="daterange"
                  size="small"
                  value-format="yyyy-MM-dd"
                  range-separator="至"
                  start-placeholder="起"
                  end-placeholder="止"
                  style="width: 240px"
                />
              </el-form-item>
              <el-form-item label="排序">
                <el-select v-model="umFilters.sort_by" size="small" style="width: 150px">
                  <el-option label="开通时间 ↓" value="created_at_desc" />
                  <el-option label="开通时间 ↑" value="created_at_asc" />
                  <el-option label="到期时间 ↓" value="expire_at_desc" />
                  <el-option label="到期时间 ↑" value="expire_at_asc" />
                </el-select>
              </el-form-item>
              <el-form-item>
                <el-button type="primary" size="mini" @click="umSearch">查询</el-button>
                <el-button size="mini" @click="umResetFilters">重置</el-button>
                <el-button type="primary" size="mini" plain @click="openGrantDialogBatch">发放/调整（批量）</el-button>
                <el-button size="mini" :disabled="!umSelection.length" @click="batchUmRenew">批量续期</el-button>
                <el-button size="mini" :disabled="!umSelection.length" @click="batchUmDowngrade">批量降级</el-button>
                <el-button size="mini" :disabled="!umSelection.length" @click="batchUmFreeze">批量冻结</el-button>
                <el-button size="mini" :disabled="!umSelection.length" @click="batchUmUnfreeze">批量解冻</el-button>
                <el-button size="mini" @click="exportUmCsv">导出 CSV</el-button>
              </el-form-item>
            </el-form>
            <el-table
              v-loading="loading.um"
              :data="umList"
              border
              @selection-change="onUmSelectionChange"
            >
              <el-table-column type="selection" width="44" fixed />
              <el-table-column label="头像" width="64">
                <template slot-scope="{ row }">
                  <el-avatar v-if="row.avatar" :size="32" :src="row.avatar" />
                  <el-avatar v-else :size="32" icon="el-icon-user-solid" />
                </template>
              </el-table-column>
              <el-table-column label="用户ID" prop="user_id" width="88" />
              <el-table-column label="昵称" prop="nickname" min-width="100" show-overflow-tooltip />
              <el-table-column label="手机号" width="120">
                <template slot-scope="{ row }">{{ row.mobile_masked || '—' }}</template>
              </el-table-column>
              <el-table-column label="等级" min-width="140">
                <template slot-scope="{ row }">
                  <div>{{ row.level_name || '—' }}</div>
                  <span class="sub-text">{{ row.level_code || '—' }}</span>
                </template>
              </el-table-column>
              <el-table-column label="状态" width="100">
                <template slot-scope="{ row }">
                  <el-tag size="mini" :type="umValidityTagType(row.validity_status)">{{ umValidityLabel(row.validity_status) }}</el-tag>
                </template>
              </el-table-column>
              <el-table-column label="剩余天数" width="88">
                <template slot-scope="{ row }">
                  <span v-if="row.validity_status === 'frozen'">—</span>
                  <span v-else-if="row.is_permanent === 1">永久</span>
                  <span v-else-if="row.remaining_days != null">{{ row.remaining_days }}</span>
                  <span v-else>—</span>
                </template>
              </el-table-column>
              <el-table-column label="开通" width="108">
                <template slot-scope="{ row }">{{ formatDateTime(row.created_at) }}</template>
              </el-table-column>
              <el-table-column label="到期" width="108">
                <template slot-scope="{ row }">{{ formatExpire(row) }}</template>
              </el-table-column>
              <el-table-column label="渠道" width="96">
                <template slot-scope="{ row }">{{ registerTypeLabel(row.register_type) }}</template>
              </el-table-column>
              <el-table-column label="发放人" width="100" show-overflow-tooltip>
                <template slot-scope="{ row }">{{ row.grant_admin_name || '—' }}</template>
              </el-table-column>
              <el-table-column label="操作" width="220" fixed="right">
                <template slot-scope="{ row }">
                  <el-button size="mini" type="text" @click="openUserDetail(row)">详情</el-button>
                  <el-button size="mini" type="text" @click="openGrantDialog(row)">调整</el-button>
                  <el-button size="mini" type="text" @click="quickRenewOne(row)">续期</el-button>
                  <el-button size="mini" type="primary" @click="openBindDeviceDialog(row)">绑定设备</el-button>
                </template>
              </el-table-column>
              <template slot="empty">
                <el-empty description="暂无会员数据，可先查询或点击「发放/调整（批量）」开通会员">
                  <el-button type="primary" size="small" @click="openGrantDialogBatch">发放/调整（批量）</el-button>
                </el-empty>
              </template>
            </el-table>
            <div class="um-pager">
              <el-pagination
                background
                layout="total, sizes, prev, pager, next"
                :total="umTotal"
                :page-size="umPageSize"
                :current-page.sync="umPage"
                :page-sizes="[10, 20, 50, 100]"
                @size-change="umSizeChange"
                @current-change="umPageChange"
              />
            </div>
          </el-tab-pane>
        </el-tabs>
      </el-card>

      <!-- Level dialog -->
      <el-dialog :title="levelDialog.title" :visible.sync="levelDialog.open" width="560px" append-to-body :close-on-click-modal="false">
        <el-form ref="levelForm" :model="levelDialog.form" label-width="108px">
          <el-form-item label="编码">
            <el-input v-model="levelDialog.form.level_code" placeholder="ordinary/vip/svip" :disabled="!!levelDialog.form.id" />
          </el-form-item>
          <el-form-item label="名称">
            <el-input v-model="levelDialog.form.level_name" placeholder="普通会员/VIP/SVIP" />
          </el-form-item>
          <el-form-item label="排序">
            <el-input-number v-model="levelDialog.form.sort" :min="1" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="levelDialog.form.status" style="width: 100%">
              <el-option label="启用" :value="1" />
              <el-option label="禁用" :value="0" />
            </el-select>
          </el-form-item>
          <el-form-item label="获取方式">
            <el-select v-model="levelDialog.form.acquire_type" style="width: 100%">
              <el-option label="注册即有" :value="1" />
              <el-option label="付费购买" :value="2" />
              <el-option label="活动赠送" :value="3" />
              <el-option label="邀请解锁" :value="4" />
            </el-select>
          </el-form-item>
          <el-form-item label="默认有效天数">
            <el-input
              v-model="levelDialog.form.default_validity_input"
              clearable
              placeholder="可选，正整数天数；留空表示不按固定天数展示"
            />
            <div class="form-hint">留空则后端不写入固定默认天数（实际以订单为准）。</div>
          </el-form-item>
          <el-form-item label="价格/套餐文案">
            <el-input v-model="levelDialog.form.price_package_hint" type="textarea" :rows="2" placeholder="如：月付 ¥18 / 年付 ¥168" />
          </el-form-item>
          <el-form-item label="版本号">
            <el-input-number v-model="levelDialog.form.version" :min="1" :max="9999" />
          </el-form-item>
          <el-form-item label="扩展 JSON">
            <el-input
              v-model="levelDialog.form.extra_config_json"
              type="textarea"
              :rows="4"
              placeholder="升级规则、备注等 JSON，可为空对象 {}"
            />
          </el-form-item>
        </el-form>
        <div slot="footer" class="dialog-footer">
          <el-button type="primary" @click="submitLevel">保存</el-button>
          <el-button @click="levelDialog.open = false">取消</el-button>
        </div>
      </el-dialog>

      <!-- Benefit dialog -->
      <el-dialog :title="benefitDialog.title" :visible.sync="benefitDialog.open" width="640px" append-to-body :close-on-click-modal="false">
        <el-form ref="benefitForm" :model="benefitDialog.form" label-width="108px">
          <el-form-item label="编码">
            <el-input v-model="benefitDialog.form.benefit_code" placeholder="play_high_rate" />
          </el-form-item>
          <el-form-item label="名称">
            <el-input v-model="benefitDialog.form.benefit_name" placeholder="高码率播放" />
          </el-form-item>
          <el-form-item label="说明">
            <el-input v-model="benefitDialog.form.description" type="textarea" :rows="3" />
          </el-form-item>
          <el-form-item label="类型">
            <el-select v-model="benefitDialog.form.benefit_type" style="width: 100%">
              <el-option label="基础功能" :value="1" />
              <el-option label="增值服务" :value="2" />
              <el-option label="限时活动" :value="3" />
            </el-select>
          </el-form-item>
          <el-form-item label="生命周期">
            <el-select v-model="benefitDialog.form.lifecycle_status" style="width: 100%">
              <el-option label="草稿" :value="0" />
              <el-option label="待发布" :value="1" />
              <el-option label="已上线" :value="2" />
              <el-option label="已下线" :value="3" />
            </el-select>
          </el-form-item>
          <el-form-item label="版本号">
            <el-input-number v-model="benefitDialog.form.version" :min="1" :max="999" />
          </el-form-item>
          <el-form-item label="标签">
            <el-input v-model="benefitDialog.form.tags" placeholder="逗号分隔，如 核心功能,新用户专属" />
          </el-form-item>
          <el-form-item label="生效开始">
            <el-date-picker
              v-model="benefitDialog.form.effect_start_at"
              type="datetime"
              value-format="yyyy-MM-dd HH:mm:ss"
              placeholder="可选"
              style="width: 100%"
            />
          </el-form-item>
          <el-form-item label="生效结束">
            <el-date-picker
              v-model="benefitDialog.form.effect_end_at"
              type="datetime"
              value-format="yyyy-MM-dd HH:mm:ss"
              placeholder="可选"
              style="width: 100%"
            />
          </el-form-item>
          <el-form-item label="扩展 JSON">
            <el-input
              v-model="benefitDialog.form.extra_config_json"
              type="textarea"
              :rows="5"
              placeholder="JSON：mutual_exclusive_codes / dependent_codes / limit 等"
            />
            <div class="form-hint">互斥：mutual_exclusive_codes；依赖：dependent_codes；限额等可先写在此，后续接入规则引擎。</div>
          </el-form-item>
        </el-form>
        <div slot="footer" class="dialog-footer">
          <el-button type="primary" @click="submitBenefit">保存</el-button>
          <el-button @click="benefitDialog.open = false">取消</el-button>
        </div>
      </el-dialog>

      <!-- 等级权益：配置预览 -->
      <el-dialog title="配置预览" :visible.sync="previewDialog.open" width="720px" append-to-body>
        <p class="preview-summary">等级 <strong>{{ mapping.level_code }}</strong> 已选 {{ mapping.benefit_codes.length }} 项权益，可导出为清单核对。</p>
        <el-table :data="previewRows" border size="small" max-height="360">
          <el-table-column prop="group" label="分组" width="120" />
          <el-table-column prop="benefit_name" label="权益名称" min-width="120" />
          <el-table-column prop="benefit_code" label="权益 code" width="140" />
          <el-table-column prop="description" label="说明" min-width="160" show-overflow-tooltip />
        </el-table>
        <div slot="footer" class="dialog-footer">
          <el-button size="small" @click="copyPreviewText">复制清单文本</el-button>
          <el-button type="primary" size="small" @click="previewDialog.open = false">关闭</el-button>
        </div>
      </el-dialog>

      <!-- 等级权益：与其他等级对比 -->
      <el-dialog title="配置对比" :visible.sync="compareDialog.open" width="800px" append-to-body>
        <el-form :inline="true" class="mb12">
          <el-form-item label="对比等级">
            <el-select v-model="compareDialog.otherLevel" placeholder="选择要对比的等级" size="small" style="width: 280px" filterable>
              <el-option
                v-for="l in sortedLevelsForMapping"
                :key="'cmp-' + l.level_code"
                :disabled="l.level_code === mapping.level_code"
                :label="levelOptionLabel(l)"
                :value="l.level_code"
              />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" size="mini" :disabled="!compareDialog.otherLevel" :loading="compareDialog.loading" @click="runCompare">加载对比</el-button>
          </el-form-item>
        </el-form>
        <el-row :gutter="8">
          <el-col :span="12">
            <div class="compare-col-title">当前：{{ mapping.level_code || '-' }}</div>
            <el-tag v-for="c in compareDialog.currentCodes" :key="'c-' + c" size="mini" class="compare-tag">{{ c }}</el-tag>
            <el-empty v-if="!compareDialog.currentCodes.length" description="无" :image-size="48" />
          </el-col>
          <el-col :span="12">
            <div class="compare-col-title">对比：{{ compareDialog.otherLevel || '-' }}</div>
            <el-tag v-for="o in compareDialog.otherCodes" :key="'o-' + o" size="mini" type="info" class="compare-tag">{{ o }}</el-tag>
            <el-empty v-if="compareDialog.otherLevel && !compareDialog.otherCodes.length" description="无" :image-size="48" />
          </el-col>
        </el-row>
        <div v-if="compareDiffSummary" class="compare-diff">{{ compareDiffSummary }}</div>
        <div slot="footer" class="dialog-footer">
          <el-button type="primary" size="small" @click="compareDialog.open = false">关闭</el-button>
        </div>
      </el-dialog>

      <!-- Grant dialog -->
      <el-dialog
        :title="grantDialog.batch ? '批量发放/调整会员' : '发放/调整会员'"
        :visible.sync="grantDialog.open"
        width="580px"
        append-to-body
        :close-on-click-modal="false"
      >
        <el-alert
          v-if="grantDialog.batch"
          class="mb12"
          type="info"
          :closable="false"
          show-icon
          title="可从上方表格勾选用户，或在文本框中输入用户ID（支持逗号、换行、分号分隔）。"
        />
        <el-form :model="grantDialog.form" label-width="110px">
          <el-form-item v-if="!grantDialog.batch" label="用户ID">
            <el-input v-model.number="grantDialog.form.user_id" disabled />
          </el-form-item>
          <el-form-item v-else label="用户ID">
            <el-input
              v-model="grantDialog.form.batchUserIdsText"
              type="textarea"
              :rows="5"
              placeholder="每行一个 ID，或用逗号分隔，例如：10001&#10;10002"
            />
          </el-form-item>
          <el-form-item label="等级">
            <el-select v-model="grantDialog.form.level_code" placeholder="请选择" style="width: 100%" filterable>
              <el-option v-for="l in enabledLevelsForGrant" :key="'g-'+l.level_code" :label="`${l.level_name}（${l.level_code}）`" :value="l.level_code" />
            </el-select>
          </el-form-item>
          <el-form-item label="到期时间">
            <el-date-picker v-model="grantDialog.form.expire_at" type="date" value-format="yyyy-MM-dd" placeholder="可选" style="width: 100%" />
          </el-form-item>
          <el-form-item label="永久">
            <el-switch v-model="grantDialog.form.is_permanent" />
          </el-form-item>
          <el-form-item label="发放渠道">
            <el-select v-model="grantDialog.form.register_type" placeholder="admin/pay/gift/register" style="width: 100%">
              <el-option label="人工 admin" value="admin" />
              <el-option label="自购 pay" value="pay" />
              <el-option label="活动 gift" value="gift" />
              <el-option label="注册 register" value="register" />
            </el-select>
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="grantDialog.form.status" style="width: 100%">
              <el-option label="有效" :value="1" />
              <el-option label="暂停（冻结）" :value="0" />
            </el-select>
          </el-form-item>
        </el-form>
        <div slot="footer" class="dialog-footer">
          <el-button type="primary" @click="submitGrant">保存</el-button>
          <el-button @click="grantDialog.open = false">取消</el-button>
        </div>
      </el-dialog>

      <!-- 用户会员详情抽屉 -->
      <el-drawer title="会员详情" :visible.sync="userDetailDrawer.open" size="480px" append-to-body>
        <div v-loading="userDetailDrawer.loading" class="um-detail">
          <template v-if="userDetailDrawer.data">
            <el-descriptions :column="1" border size="small" title="用户">
              <el-descriptions-item label="用户ID">{{ userDetailDrawer.data.user_id }}</el-descriptions-item>
              <el-descriptions-item label="昵称">{{ userDetailDrawer.data.nickname || '—' }}</el-descriptions-item>
              <el-descriptions-item label="手机">{{ userDetailDrawer.data.mobile_masked || '—' }}</el-descriptions-item>
            </el-descriptions>
            <el-descriptions v-if="userDetailDrawer.data.member" :column="1" border size="small" title="会员" class="mt12">
              <el-descriptions-item label="等级">{{ userDetailDrawer.data.level_name || '—' }}（{{ displayLevelCode(userDetailDrawer.data.member) }}）</el-descriptions-item>
              <el-descriptions-item label="到期">{{ formatExpireMember(userDetailDrawer.data.member) }}</el-descriptions-item>
              <el-descriptions-item label="永久">{{ userDetailDrawer.data.member.is_permanent === 1 ? '是' : '否' }}</el-descriptions-item>
              <el-descriptions-item label="渠道">{{ registerTypeLabel(userDetailDrawer.data.member.register_type) }}</el-descriptions-item>
              <el-descriptions-item label="状态">{{ userDetailDrawer.data.member.status === 1 ? '有效' : '冻结' }}</el-descriptions-item>
            </el-descriptions>
            <div v-else class="mt12 muted">该用户尚无会员记录，可通过「调整」开通。</div>
            <div class="mt12">
              <div class="detail-section-title">当前等级权益（编码）</div>
              <el-tag v-for="c in (userDetailDrawer.data.benefit_codes || [])" :key="'bd-'+c" size="mini" class="mr6 mb6">{{ c }}</el-tag>
              <span v-if="!(userDetailDrawer.data.benefit_codes && userDetailDrawer.data.benefit_codes.length)" class="muted">暂无或未配置等级权益</span>
            </div>
            <el-alert class="mt12" type="info" :closable="false" title="订单与操作审计为 P2 能力，后续可接入订单表与统一审计日志。" />
          </template>
        </div>
      </el-drawer>

      <!-- 等级权益：规则说明（扩展 JSON 解读） -->
      <el-dialog title="权益规则说明" :visible.sync="benefitRuleDrawer.open" width="520px" append-to-body>
        <div v-if="benefitRuleDrawer.row">
          <p><strong>{{ benefitRuleDrawer.row.benefit_name }}</strong>（{{ benefitRuleDrawer.row.benefit_code }}）</p>
          <p class="muted">{{ benefitDetailRule(benefitRuleDrawer.row) }}</p>
          <el-divider />
          <ul class="rule-list">
            <li v-for="(h, i) in benefitExtraHints(benefitRuleDrawer.row)" :key="'rule-h-'+i">{{ h }}</li>
          </ul>
          <p class="muted">生命周期与生效窗口：{{ formatEffectRange(benefitRuleDrawer.row) }}</p>
        </div>
        <div slot="footer" class="dialog-footer">
          <el-button type="primary" size="small" @click="benefitRuleDrawer.open = false">关闭</el-button>
        </div>
      </el-dialog>

      <!-- 绑定设备弹窗 -->
      <el-dialog
        :visible.sync="bindDeviceDialogVisible"
        title="绑定设备"
        width="500px"
        append-to-body
      >
        <el-form label-width="80px">
          <el-form-item label="用户 ID">
            <span>{{ currentUserInfo.user_id }}</span>
          </el-form-item>
          <el-form-item label="昵称">
            <span>{{ currentUserInfo.nickname }}</span>
          </el-form-item>
          <el-form-item label="设备 SN">
            <el-input
              v-model="bindDeviceForm.deviceSn"
              placeholder="请输入设备 SN"
              clearable
            />
          </el-form-item>
        </el-form>
        <div slot="footer" class="dialog-footer">
          <el-button @click="bindDeviceDialogVisible = false">取 消</el-button>
          <el-button type="primary" @click="doBindDevice">确认绑定</el-button>
        </div>
      </el-dialog>
    </template>
  </BasicLayout>
</template>

<script>
import BasicLayout from '@/layout/BasicLayout'
import {
  listMemberLevels,
  upsertMemberLevel,
  deleteMemberLevel,
  batchMemberLevels,
  reorderMemberLevels,
  listMemberBenefits,
  upsertMemberBenefit,
  deleteMemberBenefit,
  getLevelBenefits,
  setLevelBenefits,
  upsertUserMember,
  batchMemberBenefits,
  listUserMembers,
  userMemberSummary,
  getUserMemberDetail,
  batchUserMembers
} from '@/api/admin/platform-member'

const BENEFIT_OP_LOG_KEY = 'platform_member_benefit_oplog_v1'
const LEVEL_OP_LOG_KEY = 'platform_member_level_oplog_v1'

/** 权益分组（前端映射；后续可由后端 category 字段替代） */
const BENEFIT_GROUP_DEF = {
  basic: { key: 'basic', label: '基础权益', hint: '播放与体验', icon: 'el-icon-s-flag' },
  exclusive: { key: 'exclusive', label: '专属权益', hint: '下载与增值', icon: 'el-icon-star-on' },
  limited: { key: 'limited', label: '限时 / 活动权益', hint: '营销或高阶能力', icon: 'el-icon-time' },
  other: { key: 'other', label: '其他权益', hint: '自定义或未分组', icon: 'el-icon-menu' }
}
const BENEFIT_CODE_GROUP = {
  play_high_rate: 'basic',
  download_list: 'exclusive',
  ad_free: 'limited',
  ai_audio_generate: 'limited'
}
const BENEFIT_RULES = {
  play_high_rate: '高码率播放：支持更高音质/码率播放（如高码率/无损）；仅在会员有效期内生效。',
  download_list: '下载清单：允许离线/批量下载；具体条数与端能力以产品策略为准。',
  ad_free: '免广告：去除广告展示；若存在「广告分成」类权益请避免同时勾选。'
}
const BENEFIT_ICONS = {
  play_high_rate: 'el-icon-headset',
  download_list: 'el-icon-download',
  ad_free: 'el-icon-close-notification',
  ai_audio_generate: 'el-icon-microphone'
}
const CONFLICT_PAIRS = [
  ['ad_free', 'ad_revenue_share']
]

const MAPPING_LOG_KEY = 'platform_member_mapping_oplog_v1'

export default {
  name: 'PlatformMember',
  components: { BasicLayout },
  data() {
    return {
      active: 'levels',
      loading: { levels: false, benefits: false, mapping: false, um: false },
      levels: [],
      levelQuery: { keyword: '', status: '', acquireType: '' },
      levelSortBy: 'sort',
      levelSelection: [],
      levelOpLog: [],
      benefits: [],
      mapping: { level_code: '', benefit_codes: [] },
      mappingPreviousLevel: '',
      mappingSnapshot: '',
      mappingLastServerCodes: [],
      mappingUndoSnapshot: null,
      mappingOpLog: [],
      mappingInheritedCodes: [],
      mappingInheritedFromLevels: [],
      benefitRuleDrawer: { open: false, row: null },
      mappingCollapseNames: [],
      previewDialog: { open: false },
      compareDialog: { open: false, otherLevel: '', otherCodes: [], currentCodes: [], loading: false },
      levelDialog: {
        open: false,
        title: '新增等级',
        form: {
          id: 0,
          level_code: '',
          level_name: '',
          sort: 1,
          status: 1,
          acquire_type: 1,
          default_validity_input: '',
          price_package_hint: '',
          version: 1,
          extra_config_json: '{}'
        }
      },
      benefitDialog: {
        open: false,
        title: '新增权益',
        form: {
          id: 0,
          benefit_code: '',
          benefit_name: '',
          description: '',
          benefit_type: 1,
          lifecycle_status: 2,
          version: 1,
          tags: '',
          effect_start_at: '',
          effect_end_at: '',
          extra_config_json: '{}'
        }
      },
      grantDialog: {
        open: false,
        batch: false,
        form: {
          user_id: 0,
          batchUserIdsText: '',
          level_code: 'ordinary',
          expire_at: '',
          is_permanent: false,
          register_type: 'admin',
          grant_by: 0,
          status: 1
        }
      },
      umSummary: {},
      umList: [],
      umTotal: 0,
      umPage: 1,
      umPageSize: 20,
      umFilters: {
        keyword: '',
        level_code: '',
        validity_status: '',
        register_type: '',
        sort_by: 'created_at_desc',
        createdRange: null
      },
      umSelection: [],
      userDetailDrawer: { open: false, loading: false, data: null },
      benefitQuery: { keyword: '', lifecycle: '', benefitType: '' },
      benefitSelection: [],
      benefitShowMetrics: false,
      benefitOpLog: [],
      // 绑定设备相关
      bindDeviceDialogVisible: false,
      bindDeviceForm: {
        userId: 0,
        deviceSn: ''
      },
      currentUserInfo: {}
    }
  },
  computed: {
    filteredLevels() {
      const kw = (this.levelQuery.keyword || '').trim().toLowerCase()
      const st = this.levelQuery.status
      const at = this.levelQuery.acquireType
      let list = (this.levels || []).filter((row) => {
        if (st !== '' && st !== null && st !== undefined && row.status !== st) return false
        if (at !== '' && at !== null && at !== undefined) {
          const rowAt = row.acquire_type >= 1 && row.acquire_type <= 4 ? row.acquire_type : 1
          if (rowAt !== at) return false
        }
        if (!kw) return true
        const blob = [row.level_code, row.level_name, row.price_package_hint || '']
          .filter(Boolean)
          .join(' ')
          .toLowerCase()
        return blob.indexOf(kw) >= 0
      })
      const sortKey = this.levelSortBy || 'sort'
      list = [...list]
      list.sort((a, b) => {
        if (sortKey === 'user_count') {
          return (b.user_count || 0) - (a.user_count || 0) || (a.sort || 0) - (b.sort || 0)
        }
        if (sortKey === 'benefit_count') {
          return (b.benefit_count || 0) - (a.benefit_count || 0) || (a.sort || 0) - (b.sort || 0)
        }
        return (a.sort || 0) - (b.sort || 0) || (a.id || 0) - (b.id || 0)
      })
      return list
    },
    filteredBenefits() {
      const kw = (this.benefitQuery.keyword || '').trim().toLowerCase()
      const lc = this.benefitQuery.lifecycle
      const bt = this.benefitQuery.benefitType
      return (this.benefits || []).filter((row) => {
        const lifecycle = row.lifecycle_status !== undefined && row.lifecycle_status !== null
          ? row.lifecycle_status
          : (row.status === 1 ? 2 : 3)
        if (lc !== '' && lc !== null && lc !== undefined && lifecycle !== lc) return false
        const bty = row.benefit_type !== undefined && row.benefit_type !== null ? row.benefit_type : 1
        if (bt !== '' && bt !== null && bt !== undefined && bty !== bt) return false
        if (!kw) return true
        const blob = [row.benefit_code, row.benefit_name, row.description, row.tags]
          .filter(Boolean)
          .join(' ')
          .toLowerCase()
        return blob.indexOf(kw) >= 0
      })
    },
    sortedLevelsForMapping() {
      const list = (this.levels || []).slice()
      list.sort((a, b) => (b.sort || 0) - (a.sort || 0))
      return list
    },
    currentLevelRow() {
      return (this.levels || []).find((l) => l.level_code === this.mapping.level_code) || null
    },
    enabledBenefits() {
      return (this.benefits || []).filter((b) => {
        const ls = b.lifecycle_status !== undefined && b.lifecycle_status !== null
          ? b.lifecycle_status
          : (b.status === 1 ? 2 : 3)
        return ls === 2
      })
    },
    mappingSelectableBenefits() {
      return (this.benefits || []).filter((b) => this.mappingBenefitSelectable(b))
    },
    mappingSelectableExclusiveBenefits() {
      const inh = new Set(this.mappingInheritedCodes || [])
      return this.mappingSelectableBenefits.filter((b) => !inh.has(b.benefit_code))
    },
    exclusiveBenefitCodes: {
      get() {
        const inh = new Set(this.mappingInheritedCodes || [])
        return (this.mapping.benefit_codes || []).filter((c) => !inh.has(c))
      },
      set(val) {
        const inh = this.mappingInheritedCodes || []
        const merged = [...new Set([...(inh || []), ...(val || [])])]
        this.$set(this.mapping, 'benefit_codes', merged)
      }
    },
    inheritedGroupedForMapping() {
      return this.groupBenefitsFilter((b) => (this.mappingInheritedCodes || []).indexOf(b.benefit_code) >= 0)
    },
    exclusiveGroupedForMapping() {
      return this.groupBenefitsFilter((b) => (this.mappingInheritedCodes || []).indexOf(b.benefit_code) < 0)
    },
    mappingConflictWarnings() {
      const codes = new Set(this.mapping.benefit_codes || [])
      const out = []
      CONFLICT_PAIRS.forEach(([a, b]) => {
        if (codes.has(a) && codes.has(b)) {
          out.push(`互斥提示：「${a}」与「${b}」不建议同时配置，请取消其一。`)
        }
      })
      return out
    },
    canUndoMapping() {
      return !!(this.mappingUndoSnapshot && this.mappingUndoSnapshot.level_code)
    },
    previewRows() {
      const byCode = {}
      ;(this.benefits || []).forEach((b) => { byCode[b.benefit_code] = b })
      return (this.mapping.benefit_codes || []).map((code) => {
        const b = byCode[code] || { benefit_code: code, benefit_name: code, description: '' }
        const g = BENEFIT_CODE_GROUP[code] || 'other'
        return {
          group: (BENEFIT_GROUP_DEF[g] && BENEFIT_GROUP_DEF[g].label) || '其他',
          benefit_name: b.benefit_name,
          benefit_code: code,
          description: b.description || ''
        }
      })
    },
    compareDiffSummary() {
      const a = new Set(this.compareDialog.currentCodes || [])
      const b = new Set(this.compareDialog.otherCodes || [])
      if (!a.size && !b.size) return ''
      const onlyA = [...a].filter((x) => !b.has(x))
      const onlyB = [...b].filter((x) => !a.has(x))
      const parts = []
      if (onlyA.length) parts.push(`仅当前等级有：${onlyA.join('、')}`)
      if (onlyB.length) parts.push(`仅对比等级有：${onlyB.join('、')}`)
      return parts.join('；') || '两等级权益集合一致'
    },
    enabledLevelsForGrant() {
      return (this.levels || []).filter((l) => l.status === 1)
    }
  },
  created() {
    try {
      const raw = localStorage.getItem(MAPPING_LOG_KEY)
      if (raw) this.mappingOpLog = JSON.parse(raw)
    } catch (e) {
      this.mappingOpLog = []
    }
    try {
      const br = localStorage.getItem(BENEFIT_OP_LOG_KEY)
      if (br) this.benefitOpLog = JSON.parse(br)
    } catch (e) {
      this.benefitOpLog = []
    }
    try {
      const lr = localStorage.getItem(LEVEL_OP_LOG_KEY)
      if (lr) this.levelOpLog = JSON.parse(lr)
    } catch (e) {
      this.levelOpLog = []
    }
    this.reloadAll()
  },
  methods: {
    noop() {},
    acquireTypeLabel(t) {
      const v = t >= 1 && t <= 4 ? t : 1
      const m = { 1: '注册即有', 2: '付费购买', 3: '活动赠送', 4: '邀请解锁' }
      return m[v] || '注册即有'
    },
    formatDefaultValidityDays(v) {
      if (v === null || v === undefined || v === '') return '—'
      const n = parseInt(String(v), 10)
      if (isNaN(n) || n < 1) return '—'
      return `${n} 天`
    },
    onLevelSelectionChange(rows) {
      this.levelSelection = rows || []
    },
    pushLevelLog(detail) {
      const row = { time: new Date().toLocaleString(), detail }
      this.levelOpLog.unshift(row)
      this.levelOpLog = this.levelOpLog.slice(0, 20)
      try {
        localStorage.setItem(LEVEL_OP_LOG_KEY, JSON.stringify(this.levelOpLog))
      } catch (e) {
        void e
      }
    },
    batchLevelStatus(st) {
      const ids = (this.levelSelection || []).map((r) => r.id).filter((id) => id > 0)
      if (!ids.length) return
      const lab = st === 1 ? '启用' : '禁用'
      this.$confirm(`确认批量${lab} ${ids.length} 个等级？`, '批量操作', { type: 'warning' })
        .then(() => batchMemberLevels({ ids, action: 'set_status', status: st }))
        .then(() => {
          this.$message.success('已更新')
          this.pushLevelLog(`批量${lab}，数量 ${ids.length}`)
          this.loadLevels()
        })
        .catch(() => {})
    },
    exportLevelsCsv() {
      const rows = this.levelSelection.length ? this.levelSelection : this.filteredLevels
      if (!rows.length) {
        this.$message.warning('无数据可导出')
        return
      }
      const header = [
        'level_code',
        'level_name',
        'sort',
        'version',
        'status',
        'acquire_type',
        'price_package_hint',
        'default_validity_days',
        'user_count',
        'benefit_count'
      ]
      const lines = [header.join(',')]
      rows.forEach((r) => {
        const line = header.map((k) => {
          let v = r[k]
          if (k === 'acquire_type') v = this.acquireTypeLabel(r.acquire_type)
          if (v === undefined || v === null) return '""'
          const s = String(v).replace(/"/g, '""')
          return `"${s}"`
        })
        lines.push(line.join(','))
      })
      const blob = new Blob(['\ufeff' + lines.join('\n')], { type: 'text/csv;charset=utf-8' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = `member_levels_${Date.now()}.csv`
      a.click()
      URL.revokeObjectURL(a.href)
      this.pushLevelLog(`导出 CSV，${rows.length} 条`)
    },
    copyLevelFromSelection() {
      const row = this.levelSelection[0]
      if (!row) return
      const suffix = '_' + Date.now().toString(36)
      let extraStr = '{}'
      if (row.extra_config) {
        try {
          extraStr = typeof row.extra_config === 'string' ? row.extra_config : JSON.stringify(row.extra_config, null, 2)
        } catch (e) {
          extraStr = '{}'
        }
      }
      const maxSort = (this.levels || []).reduce((m, x) => Math.max(m, x.sort || 0), 0)
      this.levelDialog.title = '复制新增等级'
      this.levelDialog.form = {
        id: 0,
        level_code: (row.level_code || 'level') + suffix,
        level_name: (row.level_name || '') + '（副本）',
        sort: maxSort + 1,
        status: 0,
        acquire_type: row.acquire_type >= 1 && row.acquire_type <= 4 ? row.acquire_type : 1,
        default_validity_input: row.default_validity_days != null && row.default_validity_days !== undefined
          ? String(row.default_validity_days)
          : '',
        price_package_hint: row.price_package_hint || '',
        version: 1,
        extra_config_json: extraStr
      }
      this.levelDialog.open = true
      this.pushLevelLog(`从 ${row.level_code} 复制新建表单`)
    },
    goToLevelMapping(row) {
      if (!row || !row.level_code) return
      this.active = 'mapping'
      this.mappingPreviousLevel = row.level_code
      this.mapping.level_code = row.level_code
      this.$nextTick(() => {
        this.loadMapping()
      })
    },
    levelsOrderedBySort() {
      return [...(this.levels || [])].sort((a, b) => (a.sort || 0) - (b.sort || 0) || (a.id || 0) - (b.id || 0))
    },
    canMoveLevel(row, delta) {
      const ordered = this.levelsOrderedBySort()
      const ids = ordered.map((r) => r.id)
      const i = ids.indexOf(row.id)
      const j = i + delta
      return i >= 0 && j >= 0 && j < ids.length
    },
    moveLevelSort(row, delta) {
      if (!this.canMoveLevel(row, delta)) return
      const ordered = this.levelsOrderedBySort()
      const ids = ordered.map((r) => r.id)
      const i = ids.indexOf(row.id)
      const j = i + delta
      const next = [...ids]
      const t = next[i]
      next[i] = next[j]
      next[j] = t
      reorderMemberLevels({ ids: next })
        .then(() => {
          this.$message.success('已更新排序')
          this.pushLevelLog(`排序调整：${row.level_code} ${delta < 0 ? '上移' : '下移'}`)
          this.loadLevels()
        })
        .catch(() => {})
    },
    benefitTypeLabel(t) {
      const m = { 1: '基础', 2: '增值', 3: '限时' }
      return m[t] || '基础'
    },
    lifecycleLabel(ls) {
      const m = { 0: '草稿', 1: '待发布', 2: '已上线', 3: '已下线' }
      return m[ls] || (ls === undefined || ls === null ? '—' : String(ls))
    },
    lifecycleTagType(ls) {
      if (ls === 2) return 'success'
      if (ls === 0 || ls === 1) return 'warning'
      return 'info'
    },
    formatEffectRange(row) {
      const a = row.effect_start_at
      const b = row.effect_end_at
      if (!a && !b) return '不限'
      const fmt = (x) => {
        if (!x) return '—'
        const s = String(x)
        return s.length > 19 ? s.slice(0, 19).replace('T', ' ') : s
      }
      return `${fmt(a)} ~ ${fmt(b)}`
    },
    onBenefitSelectionChange(rows) {
      this.benefitSelection = rows || []
    },
    pushBenefitLog(detail) {
      const row = { time: new Date().toLocaleString(), detail }
      this.benefitOpLog.unshift(row)
      this.benefitOpLog = this.benefitOpLog.slice(0, 20)
      try {
        localStorage.setItem(BENEFIT_OP_LOG_KEY, JSON.stringify(this.benefitOpLog))
      } catch (e) {
        void e
      }
    },
    batchBenefitLifecycle(ls) {
      const ids = (this.benefitSelection || []).map((r) => r.id).filter((id) => id > 0)
      if (!ids.length) return
      this.$confirm(`确认将选中的 ${ids.length} 条权益生命周期设为「${this.lifecycleLabel(ls)}」？`, '批量操作', { type: 'warning' })
        .then(() => batchMemberBenefits({ ids, action: 'set_lifecycle', lifecycle_status: ls }))
        .then(() => {
          this.$message.success('已更新')
          this.pushBenefitLog(`批量生命周期 -> ${this.lifecycleLabel(ls)}，数量 ${ids.length}`)
          this.loadBenefits()
        })
        .catch(() => {})
    },
    batchBenefitType(bt) {
      const ids = (this.benefitSelection || []).map((r) => r.id).filter((id) => id > 0)
      if (!ids.length) return
      this.$confirm(`确认将选中的 ${ids.length} 条标为「${this.benefitTypeLabel(bt)}」类型？`, '批量操作', { type: 'warning' })
        .then(() => batchMemberBenefits({ ids, action: 'set_benefit_type', benefit_type: bt }))
        .then(() => {
          this.$message.success('已更新')
          this.pushBenefitLog(`批量类型 -> ${this.benefitTypeLabel(bt)}，数量 ${ids.length}`)
          this.loadBenefits()
        })
        .catch(() => {})
    },
    exportBenefitsCsv() {
      const rows = this.benefitSelection.length ? this.benefitSelection : this.filteredBenefits
      if (!rows.length) {
        this.$message.warning('无数据可导出')
        return
      }
      const header = ['benefit_code', 'benefit_name', 'benefit_type', 'lifecycle_status', 'version', 'tags', 'description']
      const lines = [header.join(',')]
      rows.forEach((r) => {
        const line = header.map((k) => {
          const v = r[k]
          if (v === undefined || v === null) return ''
          const s = String(v).replace(/"/g, '""')
          return `"${s}"`
        })
        lines.push(line.join(','))
      })
      const blob = new Blob(['\ufeff' + lines.join('\n')], { type: 'text/csv;charset=utf-8' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = `member_benefits_${Date.now()}.csv`
      a.click()
      URL.revokeObjectURL(a.href)
      this.pushBenefitLog(`导出 CSV，${rows.length} 条`)
    },
    copyBenefitFromSelection() {
      const row = this.benefitSelection[0]
      if (!row) return
      const suffix = '_' + Date.now().toString(36)
      const extra = row.extra_config
      let extraStr = '{}'
      if (extra) {
        try {
          extraStr = typeof extra === 'string' ? extra : JSON.stringify(extra, null, 2)
        } catch (e) {
          extraStr = '{}'
        }
      }
      this.benefitDialog.title = '复制新增权益'
      this.benefitDialog.form = {
        id: 0,
        benefit_code: (row.benefit_code || 'benefit') + suffix,
        benefit_name: (row.benefit_name || '') + '（副本）',
        description: row.description || '',
        benefit_type: row.benefit_type !== undefined ? row.benefit_type : 1,
        lifecycle_status: 0,
        version: 1,
        tags: row.tags || '',
        effect_start_at: '',
        effect_end_at: '',
        extra_config_json: extraStr
      }
      this.benefitDialog.open = true
      this.pushBenefitLog(`从 ${row.benefit_code} 复制新建表单`)
    },
    confirmToggleBenefitOffline(row) {
      const next = row.lifecycle_status === 2 ? 3 : 2
      const act = next === 3 ? '下线' : '上线'
      this.$confirm(`确认${act}权益「${row.benefit_name}」？`, '提示', { type: 'warning' })
        .then(() => batchMemberBenefits({ ids: [row.id], action: 'set_lifecycle', lifecycle_status: next }))
        .then(() => {
          this.$message.success('已更新')
          this.pushBenefitLog(`${act}：${row.benefit_code}`)
          this.loadBenefits()
        })
        .catch(() => {})
    },
    formatTime(v) {
      if (!v) return '-'
      return String(v).slice(0, 10)
    },
    levelOptionLabel(l) {
      return `${l.level_name}（${l.level_code}）· 排序 ${l.sort}`
    },
    levelValidityHint(l) {
      const c = l.level_code
      if (c === 'ordinary') return '默认档 · 权益随注册/基础策略'
      if (c === 'vip' || c === 'year_vip') return '付费档 · 有效期以套餐/订单为准'
      if (c === 'svip') return '高阶档 · 权益更丰富'
      return '具体有效期由会员套餐与用户订单决定'
    },
    benefitIcon(code) {
      return BENEFIT_ICONS[code] || 'el-icon-medal'
    },
    benefitDetailRule(b) {
      return BENEFIT_RULES[b.benefit_code] || (b.description ? String(b.description) : '详见权益说明与产品策略。')
    },
    isBenefitChecked(code) {
      return (this.mapping.benefit_codes || []).indexOf(code) >= 0
    },
    mappingBenefitSelectable(b) {
      const ls = b.lifecycle_status !== undefined && b.lifecycle_status !== null
        ? b.lifecycle_status
        : (b.status === 1 ? 2 : 3)
      return ls !== 3
    },
    normalizedBenefitCodes() {
      return [...(this.mapping.benefit_codes || [])].sort().join('\u0001')
    },
    isMappingDirty() {
      return this.mappingSnapshot !== this.normalizedBenefitCodes()
    },
    pushMappingLog(detail) {
      const row = {
        time: new Date().toLocaleString(),
        level: this.mapping.level_code || '-',
        detail
      }
      this.mappingOpLog.unshift(row)
      this.mappingOpLog = this.mappingOpLog.slice(0, 20)
      try {
        localStorage.setItem(MAPPING_LOG_KEY, JSON.stringify(this.mappingOpLog))
      } catch (e) {
        void e
      }
    },
    onMappingLevelChange(val) {
      const prev = this.mappingPreviousLevel
      if (prev && val !== prev && this.isMappingDirty()) {
        this.$confirm('当前等级权益尚未保存，切换将放弃未保存的修改。是否继续？', '提示', {
          confirmButtonText: '继续切换',
          cancelButtonText: '留在本页',
          type: 'warning'
        }).then(() => {
          this.mappingPreviousLevel = val
          this.loadMapping()
        }).catch(() => {
          this.$nextTick(() => { this.mapping.level_code = prev })
        })
      } else {
        this.mappingPreviousLevel = val
        this.loadMapping()
      }
    },
    groupBenefitsFilter(pred) {
      const map = { basic: [], exclusive: [], limited: [], other: [] }
      ;(this.benefits || []).forEach((b) => {
        if (!pred(b)) return
        const g = BENEFIT_CODE_GROUP[b.benefit_code] || 'other'
        if (!map[g]) map.other.push(b)
        else map[g].push(b)
      })
      return Object.keys(BENEFIT_GROUP_DEF).map((k) => ({
        ...BENEFIT_GROUP_DEF[k],
        items: map[k] || []
      }))
    },
    selectAllMappingBenefits(checked) {
      const inh = new Set(this.mappingInheritedCodes || [])
      const exclusiveCodes = this.mappingSelectableBenefits
        .map((b) => b.benefit_code)
        .filter((c) => !inh.has(c))
      if (checked) {
        this.exclusiveBenefitCodes = [...new Set([...this.exclusiveBenefitCodes, ...exclusiveCodes])]
      } else {
        this.exclusiveBenefitCodes = this.exclusiveBenefitCodes.filter((c) => !exclusiveCodes.includes(c))
      }
    },
    invertMappingBenefits() {
      const inh = new Set(this.mappingInheritedCodes || [])
      const enabled = new Set(
        this.mappingSelectableBenefits.map((b) => b.benefit_code).filter((c) => !inh.has(c))
      )
      const cur = new Set(this.exclusiveBenefitCodes)
      const next = []
      enabled.forEach((c) => {
        if (!cur.has(c)) next.push(c)
      })
      this.exclusiveBenefitCodes = next
    },
    openPreviewDialog() {
      this.previewDialog.open = true
    },
    copyPreviewText() {
      const lines = this.previewRows.map((r) => `${r.group} | ${r.benefit_name} | ${r.benefit_code} | ${r.description}`)
      const text = [`等级：${this.mapping.level_code}`, `共 ${lines.length} 项`, ...lines].join('\n')
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(text).then(() => this.$message.success('已复制到剪贴板')).catch(() => this.$message.warning('复制失败'))
      } else {
        this.$message.info(text)
      }
    },
    openCompareDialog() {
      this.compareDialog.open = true
      this.compareDialog.otherLevel = ''
      this.compareDialog.otherCodes = []
      this.compareDialog.currentCodes = [...(this.mapping.benefit_codes || [])]
    },
    runCompare() {
      if (!this.compareDialog.otherLevel) return
      this.compareDialog.loading = true
      this.compareDialog.currentCodes = [...(this.mapping.benefit_codes || [])]
      getLevelBenefits(this.compareDialog.otherLevel)
        .then((res) => {
          const d = res.data || {}
          this.compareDialog.otherCodes = d.benefit_codes || []
        })
        .finally(() => { this.compareDialog.loading = false })
    },
    confirmSaveMapping() {
      if (!this.mapping.level_code) return
      const codes = this.mapping.benefit_codes || []
      const preview = codes.length ? codes.join('、') : '（未勾选任何权益）'
      this.$confirm(
        `确认为等级「${this.mapping.level_code}」保存权益配置？\n共 ${codes.length} 项：${preview}`,
        '保存确认',
        { confirmButtonText: '保存', cancelButtonText: '取消', type: 'warning' }
      ).then(() => this.saveMapping()).catch(() => {})
    },
    undoLastMappingSave() {
      if (!this.mappingUndoSnapshot) return
      const snap = this.mappingUndoSnapshot
      this.$confirm('将服务器上的配置恢复为保存前的一版，是否继续？', '撤销上次保存', {
        type: 'warning'
      }).then(() => {
        setLevelBenefits(snap.level_code, snap.codes).then(() => {
          this.mappingUndoSnapshot = null
          this.$message.success('已恢复上一版并同步到服务器')
          this.pushMappingLog(`撤销：已恢复等级 ${snap.level_code} 上一版配置`)
          this.loadMapping()
        }).catch(() => {})
      }).catch(() => {})
    },
    reloadAll() {
      this.loadLevels()
      this.loadBenefits()
      this.loadUmSummary()
      this.loadUserMembers()
    },
    loadLevels() {
      this.loading.levels = true
      listMemberLevels()
        .then((res) => { this.levels = (res.data && res.data.list) || [] })
        .finally(() => { this.loading.levels = false })
    },
    loadBenefits() {
      this.loading.benefits = true
      listMemberBenefits()
        .then((res) => { this.benefits = (res.data && res.data.list) || [] })
        .finally(() => { this.loading.benefits = false })
    },
    openLevelDialog(row) {
      if (row) {
        this.levelDialog.title = '编辑等级'
        let extraStr = '{}'
        if (row.extra_config) {
          try {
            extraStr = typeof row.extra_config === 'string'
              ? row.extra_config
              : JSON.stringify(row.extra_config, null, 2)
          } catch (e) {
            extraStr = '{}'
          }
        }
        this.levelDialog.form = {
          id: row.id,
          level_code: row.level_code,
          level_name: row.level_name,
          sort: row.sort || 1,
          status: row.status,
          acquire_type: row.acquire_type >= 1 && row.acquire_type <= 4 ? row.acquire_type : 1,
          default_validity_input: row.default_validity_days != null && row.default_validity_days !== undefined
            ? String(row.default_validity_days)
            : '',
          price_package_hint: row.price_package_hint || '',
          version: row.version !== undefined && row.version !== null ? row.version : 1,
          extra_config_json: extraStr
        }
      } else {
        const maxSort = (this.levels || []).reduce((m, x) => Math.max(m, x.sort || 0), 0)
        this.levelDialog.title = '新增等级'
        this.levelDialog.form = {
          id: 0,
          level_code: '',
          level_name: '',
          sort: maxSort + 1,
          status: 1,
          acquire_type: 1,
          default_validity_input: '',
          price_package_hint: '',
          version: 1,
          extra_config_json: '{}'
        }
      }
      this.levelDialog.open = true
    },
    submitLevel() {
      let extraObj = {}
      try {
        extraObj = JSON.parse(this.levelDialog.form.extra_config_json || '{}')
        if (typeof extraObj !== 'object' || extraObj === null) {
          throw new Error('invalid')
        }
      } catch (e) {
        this.$message.error('扩展 JSON 格式不正确')
        return
      }
      const s = (this.levelDialog.form.default_validity_input || '').trim()
      let defaultValidityDays = null
      if (s) {
        const n = parseInt(s, 10)
        if (Number.isNaN(n) || n < 1) {
          this.$message.error('默认有效天数须为正整数')
          return
        }
        defaultValidityDays = n
      }
      const existing = this.levelDialog.form.id
        ? (this.levels || []).find((l) => l.id === this.levelDialog.form.id)
        : null
      const doSave = () => {
        const payload = {
          id: this.levelDialog.form.id,
          level_code: this.levelDialog.form.level_code,
          level_name: this.levelDialog.form.level_name,
          sort: this.levelDialog.form.sort,
          status: this.levelDialog.form.status,
          acquire_type: this.levelDialog.form.acquire_type,
          default_validity_days: defaultValidityDays,
          price_package_hint: this.levelDialog.form.price_package_hint || '',
          version: this.levelDialog.form.version,
          extra_config: extraObj
        }
        upsertMemberLevel(payload)
          .then(() => {
            this.$message.success('已保存')
            this.pushLevelLog(`保存等级：${payload.level_code}，状态 ${payload.status === 1 ? '启用' : '禁用'}`)
            this.levelDialog.open = false
            this.loadLevels()
          })
          .catch(() => {})
      }
      if (this.levelDialog.form.status === 1 && existing && (existing.benefit_count === 0 || existing.benefit_count === undefined)) {
        this.$confirm(
          '当前等级权益数为 0，确定以启用状态保存？建议先在「等级权益配置」中勾选权益。',
          '提示',
          { type: 'warning', confirmButtonText: '仍要保存', cancelButtonText: '取消' }
        )
          .then(() => doSave())
          .catch(() => {})
        return
      }
      doSave()
    },
    deleteLevel(row) {
      this.$confirm(
        `确认删除等级「${row.level_name}」（${row.level_code}）？若仍有有效用户或套餐关联将无法删除。`,
        '删除确认',
        { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
      )
        .then(() => deleteMemberLevel(row.id))
        .then(() => {
          this.$message.success('已删除')
          this.pushLevelLog(`删除等级：${row.level_code}`)
          this.loadLevels()
        })
        .catch((err) => {
          if (err === 'cancel') return
          const d = err && err.response && err.response.data
          const msg = d && (d.msg || d.message)
          if (msg) this.$message.error(msg)
        })
    },

    openBenefitDialog(row) {
      if (row) {
        this.benefitDialog.title = '编辑权益'
        let extraStr = '{}'
        if (row.extra_config) {
          try {
            extraStr = typeof row.extra_config === 'string'
              ? row.extra_config
              : JSON.stringify(row.extra_config, null, 2)
          } catch (e) {
            extraStr = '{}'
          }
        }
        const es = row.effect_start_at ? String(row.effect_start_at).slice(0, 19).replace('T', ' ') : ''
        const ee = row.effect_end_at ? String(row.effect_end_at).slice(0, 19).replace('T', ' ') : ''
        this.benefitDialog.form = {
          id: row.id,
          benefit_code: row.benefit_code,
          benefit_name: row.benefit_name,
          description: row.description || '',
          benefit_type: row.benefit_type !== undefined && row.benefit_type !== null ? row.benefit_type : 1,
          lifecycle_status: row.lifecycle_status !== undefined && row.lifecycle_status !== null
            ? row.lifecycle_status
            : (row.status === 1 ? 2 : 3),
          version: row.version !== undefined && row.version !== null ? row.version : 1,
          tags: row.tags || '',
          effect_start_at: es,
          effect_end_at: ee,
          extra_config_json: extraStr
        }
      } else {
        this.benefitDialog.title = '新增权益'
        this.benefitDialog.form = {
          id: 0,
          benefit_code: '',
          benefit_name: '',
          description: '',
          benefit_type: 1,
          lifecycle_status: 2,
          version: 1,
          tags: '',
          effect_start_at: '',
          effect_end_at: '',
          extra_config_json: '{}'
        }
      }
      this.benefitDialog.open = true
    },
    submitBenefit() {
      let extraObj = {}
      try {
        extraObj = JSON.parse(this.benefitDialog.form.extra_config_json || '{}')
        if (typeof extraObj !== 'object' || extraObj === null) {
          throw new Error('invalid')
        }
      } catch (e) {
        this.$message.error('扩展 JSON 格式不正确')
        return
      }
      const mutual = extraObj.mutual_exclusive_codes
      const dep = extraObj.dependent_codes
      if (Array.isArray(mutual) && mutual.length && Array.isArray(dep) && dep.length) {
        const overlap = mutual.filter((c) => dep.indexOf(c) >= 0)
        if (overlap.length) {
          this.$message.warning('互斥列表与依赖列表存在重叠，请检查 JSON')
          return
        }
      }
      const payload = {
        id: this.benefitDialog.form.id,
        benefit_code: this.benefitDialog.form.benefit_code,
        benefit_name: this.benefitDialog.form.benefit_name,
        description: this.benefitDialog.form.description,
        benefit_type: this.benefitDialog.form.benefit_type,
        lifecycle_status: this.benefitDialog.form.lifecycle_status,
        version: this.benefitDialog.form.version,
        tags: this.benefitDialog.form.tags,
        effect_start_at: this.benefitDialog.form.effect_start_at || null,
        effect_end_at: this.benefitDialog.form.effect_end_at || null,
        extra_config: extraObj
      }
      upsertMemberBenefit(payload)
        .then(() => {
          this.$message.success('已保存')
          this.benefitDialog.open = false
          this.pushBenefitLog(`保存权益：${payload.benefit_code}，生命周期 ${this.lifecycleLabel(payload.lifecycle_status)}`)
          this.loadBenefits()
        })
        .catch(() => {})
    },
    deleteBenefit(row) {
      this.$confirm(
        `确定删除权益「${row.benefit_name}」（${row.benefit_code}）？删除后不可恢复，且可能影响等级权益配置。`,
        '删除确认',
        { type: 'warning', confirmButtonText: '删除', cancelButtonText: '取消' }
      )
        .then(() => deleteMemberBenefit(row.id))
        .then(() => {
          this.$message.success('已删除')
          this.pushBenefitLog(`删除权益：${row.benefit_code}`)
          this.loadBenefits()
        })
        .catch(() => {})
    },
    loadMapping() {
      if (!this.mapping.level_code) return
      this.loading.mapping = true
      getLevelBenefits(this.mapping.level_code)
        .then((res) => {
          const d = res.data || {}
          this.mappingInheritedCodes = d.inherited_benefit_codes || []
          this.mappingInheritedFromLevels = d.inherited_from_levels || []
          const codes = d.benefit_codes || []
          this.mapping.benefit_codes = [...new Set([...this.mappingInheritedCodes, ...codes])]
          this.mappingLastServerCodes = [...(this.mapping.benefit_codes || [])]
          this.mappingSnapshot = this.normalizedBenefitCodes()
        })
        .catch(() => {
          this.$message.error('加载等级权益失败')
        })
        .finally(() => { this.loading.mapping = false })
    },
    cancelMappingEdits() {
      if (!this.mapping.level_code) return
      this.loadMapping()
    },
    openMappingLogPanel() {
      this.mappingCollapseNames = ['log']
      this.$nextTick(() => {
        const ref = this.$refs.mappingLogCollapse
        const el = ref && (ref.$el || ref)
        if (el && el.scrollIntoView) {
          el.scrollIntoView({ behavior: 'smooth', block: 'nearest' })
        }
      })
    },
    benefitExtraHints(b) {
      const ex = b && b.extra_config
      let o = {}
      if (ex) {
        try {
          o = typeof ex === 'string' ? JSON.parse(ex) : ex
        } catch (e) {
          o = {}
        }
      }
      if (!o || typeof o !== 'object') o = {}
      const out = []
      if (o.monthly_limit != null) out.push(`次数限制：每月 ${o.monthly_limit} 次`)
      if (o.daily_limit != null) out.push(`次数限制：每日 ${o.daily_limit} 次`)
      if (Array.isArray(o.mutual_exclusive_codes) && o.mutual_exclusive_codes.length) {
        out.push(`互斥规则：与 ${o.mutual_exclusive_codes.join('、')} 不可同时配置`)
      }
      if (o.scope) out.push(`生效范围：${o.scope}`)
      if (o.effective) out.push(`生效方式：${o.effective}`)
      if (o.limit_desc) out.push(String(o.limit_desc))
      return out.length ? out : ['未在扩展 JSON 中配置细则；可在「会员权益」中编辑 extra_config（如 monthly_limit、mutual_exclusive_codes、scope）。']
    },
    openBenefitRuleDrawer(row) {
      this.benefitRuleDrawer.row = row
      this.benefitRuleDrawer.open = true
    },
    goEditBenefitFromMapping(row) {
      if (!row) return
      this.active = 'benefits'
      this.openBenefitDialog(row)
    },
    saveMapping() {
      if (!this.mapping.level_code) return
      const undoSnapshot = {
        level_code: this.mapping.level_code,
        codes: [...this.mappingLastServerCodes]
      }
      setLevelBenefits(this.mapping.level_code, this.mapping.benefit_codes)
        .then(() => {
          this.mappingUndoSnapshot = undoSnapshot
          this.mappingLastServerCodes = [...(this.mapping.benefit_codes || [])]
          this.mappingSnapshot = this.normalizedBenefitCodes()
          this.$message.success('保存成功')
          this.pushMappingLog(`保存：等级 ${this.mapping.level_code}，共 ${(this.mapping.benefit_codes || []).length} 项权益`)
        })
        .catch(() => {
          this.$message.error('保存失败，请重试')
        })
    },
    loadUmSummary() {
      userMemberSummary()
        .then((res) => {
          const d = res.data || res
          this.umSummary = d || {}
        })
        .catch(() => {})
    },
    loadUserMembers() {
      this.loading.um = true
      const f = this.umFilters
      const params = {
        page: this.umPage,
        page_size: this.umPageSize,
        sort_by: f.sort_by || 'created_at_desc'
      }
      if (f.keyword && String(f.keyword).trim()) params.keyword = String(f.keyword).trim()
      if (f.level_code) params.level_code = f.level_code
      if (f.validity_status) params.validity_status = f.validity_status
      if (f.register_type) params.register_type = f.register_type
      if (f.createdRange && f.createdRange.length === 2) {
        params.created_from = f.createdRange[0]
        params.created_to = f.createdRange[1]
      }
      listUserMembers(params)
        .then((res) => {
          const d = res.data || {}
          this.umList = d.list || []
          this.umTotal = d.total != null ? d.total : 0
        })
        .finally(() => {
          this.loading.um = false
        })
    },
    umSearch() {
      this.umPage = 1
      this.loadUserMembers()
    },
    umResetFilters() {
      this.umFilters = {
        keyword: '',
        level_code: '',
        validity_status: '',
        register_type: '',
        sort_by: 'created_at_desc',
        createdRange: null
      }
      this.umPage = 1
      this.loadUserMembers()
    },
    umPageChange(p) {
      this.umPage = p
      this.loadUserMembers()
    },
    umSizeChange(s) {
      this.umPageSize = s
      this.umPage = 1
      this.loadUserMembers()
    },
    onUmSelectionChange(rows) {
      this.umSelection = rows || []
    },
    umValidityLabel(v) {
      const m = { active: '生效中', expired: '已过期', expiring_soon: '即将到期', frozen: '冻结' }
      return m[v] || v || '—'
    },
    umValidityTagType(v) {
      if (v === 'active') return 'success'
      if (v === 'expiring_soon') return 'warning'
      if (v === 'expired') return 'info'
      if (v === 'frozen') return 'danger'
      return ''
    },
    registerTypeLabel(rt) {
      if (!rt) return '—'
      const m = { pay: '自购', gift: '活动', admin: '人工', register: '注册' }
      return m[String(rt)] || String(rt)
    },
    formatDateTime(v) {
      if (!v) return '—'
      const s = String(v)
      return s.length > 19 ? s.slice(0, 19).replace('T', ' ') : s
    },
    formatExpire(row) {
      if (!row) return '—'
      if (row.is_permanent === 1) return '永久'
      if (!row.expire_at) return '—'
      return String(row.expire_at).slice(0, 10)
    },
    formatExpireMember(m) {
      if (!m) return '—'
      if (m.is_permanent === 1) return '永久'
      if (!m.expire_at) return '—'
      return String(m.expire_at).slice(0, 10)
    },
    displayLevelCode(m) {
      if (!m) return 'ordinary'
      if (m.level_code) return m.level_code
      return 'ordinary'
    },
    openUserDetail(row) {
      if (!row || !row.user_id) return
      this.userDetailDrawer.open = true
      this.userDetailDrawer.loading = true
      this.userDetailDrawer.data = null
      getUserMemberDetail(row.user_id)
        .then((res) => {
          this.userDetailDrawer.data = res.data || res
        })
        .catch(() => {})
        .finally(() => {
          this.userDetailDrawer.loading = false
        })
    },
    resolveGrantUserIds() {
      if (this.grantDialog.batch) {
        if (this.umSelection.length) {
          return this.umSelection.map((r) => r.user_id).filter((id) => id > 0)
        }
        const t = this.grantDialog.form.batchUserIdsText || ''
        const parts = String(t).split(/[\s,，;；]+/).map((s) => s.trim()).filter(Boolean)
        const ids = []
        parts.forEach((p) => {
          const n = parseInt(p, 10)
          if (!Number.isNaN(n) && n > 0) ids.push(n)
        })
        return [...new Set(ids)]
      }
      const uid = this.grantDialog.form.user_id
      return uid > 0 ? [uid] : []
    },
    openGrantDialog(row) {
      this.grantDialog.batch = false
      const m = row || {}
      const lc = m.level_code || 'ordinary'
      this.grantDialog.form = {
        user_id: m.user_id,
        batchUserIdsText: '',
        level_code: lc,
        expire_at: m.expire_at ? String(m.expire_at).slice(0, 10) : '',
        is_permanent: m.is_permanent === 1,
        register_type: m.register_type || 'admin',
        grant_by: 0,
        status: m.status !== undefined ? m.status : 1
      }
      this.grantDialog.open = true
    },
    openGrantDialogBatch() {
      this.grantDialog.batch = true
      this.grantDialog.form = {
        user_id: 0,
        batchUserIdsText: this.umSelection.map((r) => r.user_id).join('\n'),
        level_code: 'ordinary',
        expire_at: '',
        is_permanent: false,
        register_type: 'admin',
        grant_by: 0,
        status: 1
      }
      this.grantDialog.open = true
    },
    submitGrant() {
      const ids = this.resolveGrantUserIds()
      if (!ids.length) {
        this.$message.warning('请填写用户ID或勾选表格中的用户')
        return
      }
      const run = () => {
        if (this.grantDialog.batch) {
          const body = {
            user_ids: ids,
            action: 'set_level',
            level_code: this.grantDialog.form.level_code,
            is_permanent: this.grantDialog.form.is_permanent,
            register_type: this.grantDialog.form.register_type,
            grant_by: this.grantDialog.form.grant_by || 0,
            status: this.grantDialog.form.status
          }
          if (this.grantDialog.form.expire_at) body.expire_at = this.grantDialog.form.expire_at
          batchUserMembers(body)
            .then(() => {
              this.$message.success('已保存')
              this.grantDialog.open = false
              this.loadUserMembers()
              this.loadUmSummary()
            })
            .catch(() => {})
        } else {
          const payload = { ...this.grantDialog.form }
          delete payload.batchUserIdsText
          if (!payload.expire_at) delete payload.expire_at
          upsertUserMember(payload)
            .then(() => {
              this.$message.success('已保存')
              this.grantDialog.open = false
              this.loadUserMembers()
              this.loadUmSummary()
            })
            .catch(() => {})
        }
      }
      const high = ['svip', 'year_vip']
      if (high.indexOf(this.grantDialog.form.level_code) >= 0) {
        this.$confirm('目标等级较高，确认保存？', '确认', { type: 'warning' }).then(run).catch(() => {})
        return
      }
      run()
    },
    batchUmRenew() {
      const ids = this.umSelection.map((r) => r.user_id).filter((id) => id > 0)
      if (!ids.length) return
      this.$prompt('请输入为选中用户续期的天数（正整数）', '批量续期', {
        inputPattern: /^[1-9]\d*$/,
        inputErrorMessage: '请输入正整数天数'
      })
        .then(({ value }) => {
          const days = parseInt(value, 10)
          return batchUserMembers({ user_ids: ids, action: 'renew_days', renew_days: days })
        })
        .then(() => {
          this.$message.success('已续期')
          this.loadUserMembers()
          this.loadUmSummary()
        })
        .catch(() => {})
    },
    batchUmDowngrade() {
      const ids = this.umSelection.map((r) => r.user_id).filter((id) => id > 0)
      if (!ids.length) return
      this.$confirm(`确认将 ${ids.length} 个用户批量降级为普通会员（ordinary）？`, '批量降级', { type: 'warning' })
        .then(() => batchUserMembers({ user_ids: ids, action: 'downgrade', grant_by: 0 }))
        .then(() => {
          this.$message.success('已降级')
          this.loadUserMembers()
          this.loadUmSummary()
        })
        .catch(() => {})
    },
    batchUmFreeze() {
      const ids = this.umSelection.map((r) => r.user_id).filter((id) => id > 0)
      if (!ids.length) return
      this.$confirm(`确认冻结 ${ids.length} 个用户的会员（权益暂停）？`, '批量冻结', { type: 'warning' })
        .then(() => batchUserMembers({ user_ids: ids, action: 'set_status', status: 0 }))
        .then(() => {
          this.$message.success('已冻结')
          this.loadUserMembers()
          this.loadUmSummary()
        })
        .catch(() => {})
    },
    batchUmUnfreeze() {
      const ids = this.umSelection.map((r) => r.user_id).filter((id) => id > 0)
      if (!ids.length) return
      this.$confirm(`确认解冻 ${ids.length} 个用户的会员？`, '批量解冻', { type: 'warning' })
        .then(() => batchUserMembers({ user_ids: ids, action: 'set_status', status: 1 }))
        .then(() => {
          this.$message.success('已解冻')
          this.loadUserMembers()
          this.loadUmSummary()
        })
        .catch(() => {})
    },
    quickRenewOne(row) {
      if (!row || !row.user_id) return
      this.$prompt(`为用户 ${row.user_id} 续期天数`, '续期', {
        inputPattern: /^[1-9]\d*$/,
        inputErrorMessage: '请输入正整数天数'
      })
        .then(({ value }) =>
          batchUserMembers({ user_ids: [row.user_id], action: 'renew_days', renew_days: parseInt(value, 10) })
        )
        .then(() => {
          this.$message.success('已续期')
          this.loadUserMembers()
          this.loadUmSummary()
        })
        .catch(() => {})
    },
    exportUmCsv() {
      const rows = this.umSelection.length ? this.umSelection : this.umList
      if (!rows.length) {
        this.$message.warning('无数据可导出')
        return
      }
      const header = [
        'user_id',
        'nickname',
        'mobile_masked',
        'level_name',
        'level_code',
        'validity_status',
        'remaining_days',
        'created_at',
        'expire_at',
        'register_type',
        'grant_admin_name',
        'status'
      ]
      const lines = [header.join(',')]
      rows.forEach((r) => {
        const line = header.map((k) => {
          let v = r[k]
          if (k === 'level_code') v = r.level_code || ''
          if (k === 'remaining_days') v = r.remaining_days != null ? r.remaining_days : ''
          if (k === 'status') v = r.status === 1 ? '有效' : '冻结'
          if (v === undefined || v === null) v = ''
          const s = String(v).replace(/"/g, '""')
          return `"${s}"`
        })
        lines.push(line.join(','))
      })
      const blob = new Blob(['\ufeff' + lines.join('\n')], { type: 'text/csv;charset=utf-8' })
      const a = document.createElement('a')
      a.href = URL.createObjectURL(blob)
      a.download = `user_members_${Date.now()}.csv`
      a.click()
      URL.revokeObjectURL(a.href)
    },
    /** 打开绑定设备弹窗 */
    openBindDeviceDialog(row) {
      this.currentUserInfo = {
        user_id: row.user_id,
        nickname: row.nickname || '—'
      }
      this.bindDeviceForm = {
        userId: row.user_id,
        deviceSn: ''
      }
      this.bindDeviceDialogVisible = true
    },
    /** 执行绑定设备 */
    async doBindDevice() {
      if (!this.bindDeviceForm.deviceSn) {
        this.$message.error('请输入设备 SN')
        return
      }
      try {
        const { createBind } = await import('@/api/admin/platform-user-device-bind')
        const res = await createBind(this.bindDeviceForm)
        if (res.code === 200) {
          this.$message.success('绑定成功')
          this.bindDeviceDialogVisible = false
          this.umSearch()
        } else {
          this.$message.error(res.msg || '绑定失败')
        }
      } catch (error) {
        this.$message.error('绑定失败：' + (error.message || '未知错误'))
      }
    }
  }
}
</script>

<style scoped>
.toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.title {
  font-size: 14px;
  font-weight: 600;
}
.mb12 {
  margin-bottom: 12px;
}

.mapping-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 16px;
}
.mapping-toolbar__left {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.mapping-label {
  font-size: 13px;
  color: #606266;
}
.mapping-level-select {
  min-width: 320px;
}
.level-option__title {
  font-weight: 600;
  color: #303133;
}
.level-option__sub {
  font-size: 12px;
  color: #909399;
  margin-top: 2px;
}
.mapping-current-hint {
  font-size: 12px;
  color: #67c23a;
}
.mapping-toolbar__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}

.benefit-checkbox-group {
  display: block;
  width: 100%;
}
.benefit-mapping-body {
  min-height: 120px;
}
.benefit-group {
  margin-bottom: 20px;
}
.benefit-group__title {
  font-size: 14px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.benefit-group__hint {
  font-size: 12px;
  font-weight: normal;
  color: #909399;
}
.benefit-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.benefit-card {
  border: 1px solid #ebeef5;
  border-radius: 8px;
  padding: 10px 12px;
  background: #fafafa;
  transition: border-color 0.2s, box-shadow 0.2s;
}
.benefit-card:hover {
  border-color: #c6e2ff;
  box-shadow: 0 2px 8px rgba(64, 158, 255, 0.12);
}
.benefit-card.is-configured {
  border-color: #b3e19d;
  background: #f0f9eb;
}
.benefit-card.is-offline {
  opacity: 0.55;
  background: #f4f4f5;
}
.benefit-card__inner {
  padding-top: 2px;
}
.benefit-card__icon {
  margin-right: 6px;
  color: #409eff;
}
.benefit-card__name {
  font-weight: 600;
  color: #303133;
}
.benefit-card__code {
  font-size: 12px;
  color: #909399;
  margin: 6px 0 0 22px;
  font-family: Consolas, monospace;
}
.benefit-card__desc {
  font-size: 12px;
  color: #606266;
  margin: 4px 0 0 22px;
  line-height: 1.4;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.benefit-tooltip__title {
  font-weight: 600;
  margin-bottom: 6px;
}
.benefit-tooltip__desc {
  margin-top: 8px;
  opacity: 0.9;
}

.preview-summary {
  margin: 0 0 12px;
  font-size: 13px;
  color: #606266;
}
.compare-col-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
}
.compare-tag {
  margin: 0 6px 6px 0;
}
.compare-diff {
  margin-top: 12px;
  padding: 10px;
  background: #fdf6ec;
  border-radius: 4px;
  font-size: 13px;
  color: #e6a23c;
}
.mapping-meta {
  margin-top: 20px;
}
.form-hint {
  font-size: 12px;
  color: #909399;
  margin-top: 4px;
  line-height: 1.4;
}
.benefit-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
}

.um-summary {
  margin-top: 4px;
}
.um-stat {
  background: #f5f7fa;
  border-radius: 8px;
  padding: 10px 12px;
  margin-bottom: 4px;
}
.um-stat__label {
  display: block;
  font-size: 12px;
  color: #909399;
  margin-bottom: 4px;
}
.um-stat__val {
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}
.um-stat__ok {
  color: #67c23a;
}
.um-stat__warn {
  color: #e6a23c;
}
.um-pager {
  margin-top: 12px;
  text-align: right;
}
.sub-text {
  font-size: 12px;
  color: #909399;
}
.mt12 {
  margin-top: 12px;
}
.muted {
  color: #909399;
  font-size: 13px;
}
.detail-section-title {
  font-size: 13px;
  font-weight: 600;
  margin-bottom: 8px;
  color: #303133;
}
.um-detail {
  padding: 0 8px 16px;
}
.mr6 {
  margin-right: 6px;
}
.mb6 {
  margin-bottom: 6px;
}

.mapping-section {
  margin-bottom: 24px;
  padding: 12px 0;
  border-bottom: 1px solid #ebeef5;
}
.mapping-section--exclusive {
  border-bottom: none;
}
.mapping-section__title {
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  margin-bottom: 10px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.mapping-section__icon {
  color: #409eff;
}
.mapping-inherit-hint {
  font-size: 13px;
  color: #606266;
  margin: 0 0 12px;
  line-height: 1.5;
}
.benefit-card.is-inherited {
  background: #f4f4f5;
  border-color: #dcdfe6;
  opacity: 0.95;
}
.benefit-inherited-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.benefit-lock {
  color: #909399;
}
.benefit-card__meta {
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
  line-height: 1.5;
}
.benefit-meta-line {
  display: block;
}
.benefit-card__actions {
  margin-top: 8px;
}
.mapping-footer-bar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 0;
  margin-top: 8px;
  border-top: 1px solid #ebeef5;
}
.rule-list {
  margin: 0;
  padding-left: 18px;
  color: #606266;
  line-height: 1.7;
}
</style>

