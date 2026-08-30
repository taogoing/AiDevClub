<template>
  <div class="resources-view">
    <h2>资源审核</h2>
    <el-tabs v-model="activeTab" @tab-change="handleTabChange">
      <el-tab-pane label="Skills" name="skill">
        <el-form :inline="true" @submit.prevent="loadSkills">
          <el-form-item label="关键词">
            <el-input v-model="skillKeyword" placeholder="名称/描述" clearable @clear="loadSkills" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="skillStatus" clearable placeholder="待审核" @change="loadSkills">
              <el-option label="待审核" value="pending_review" />
              <el-option label="已发布" value="published" />
              <el-option label="已拒绝" value="rejected" />
              <el-option label="已下架" value="archived" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadSkills">搜索</el-button>
          </el-form-item>
        </el-form>
        <el-table :data="skills" v-loading="skillLoading" stripe>
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="名称" show-overflow-tooltip />
          <el-table-column label="作者" width="120">
            <template #default="{ row }">{{ row.author.nickname }}</template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)">{{ getStatusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button size="small" @click="viewSkill(row)">查看</el-button>
              <el-button v-if="row.status === 'pending_review'" size="small" type="success" @click="reviewResource('skill', row.id, true)">通过</el-button>
              <el-button v-if="row.status === 'pending_review'" size="small" type="danger" @click="showRejectDialog('skill', row.id)">拒绝</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination
          v-if="skillTotal > 0"
          style="margin-top: 20px; justify-content: flex-end"
          :current-page="skillPage"
          :page-size="skillPageSize"
          :total="skillTotal"
          layout="total, prev, pager, next"
          @current-change="handleSkillPageChange"
        />
      </el-tab-pane>
      <el-tab-pane label="MCP Servers" name="mcp">
        <el-form :inline="true" @submit.prevent="loadMCPServers">
          <el-form-item label="关键词">
            <el-input v-model="mcpKeyword" placeholder="名称/描述" clearable @clear="loadMCPServers" />
          </el-form-item>
          <el-form-item label="状态">
            <el-select v-model="mcpStatus" clearable placeholder="待审核" @change="loadMCPServers">
              <el-option label="待审核" value="pending_review" />
              <el-option label="已发布" value="published" />
              <el-option label="已拒绝" value="rejected" />
              <el-option label="已下架" value="archived" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="loadMCPServers">搜索</el-button>
          </el-form-item>
        </el-form>
        <el-table :data="mcpServers" v-loading="mcpLoading" stripe>
          <el-table-column prop="id" label="ID" width="80" />
          <el-table-column prop="name" label="名称" show-overflow-tooltip />
          <el-table-column label="作者" width="120">
            <template #default="{ row }">{{ row.author.nickname }}</template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <el-tag :type="getStatusType(row.status)">{{ getStatusText(row.status) }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="200">
            <template #default="{ row }">
              <el-button size="small" @click="viewMCPServer(row)">查看</el-button>
              <el-button v-if="row.status === 'pending_review'" size="small" type="success" @click="reviewResource('mcp_server', row.id, true)">通过</el-button>
              <el-button v-if="row.status === 'pending_review'" size="small" type="danger" @click="showRejectDialog('mcp_server', row.id)">拒绝</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-pagination
          v-if="mcpTotal > 0"
          style="margin-top: 20px; justify-content: flex-end"
          :current-page="mcpPage"
          :page-size="mcpPageSize"
          :total="mcpTotal"
          layout="total, prev, pager, next"
          @current-change="handleMcpPageChange"
        />
      </el-tab-pane>
    </el-tabs>
    <el-drawer v-model="drawerVisible" :title="drawerTitle" size="60%">
      <div v-if="selectedSkill" v-loading="detailLoading">
        <h3>{{ selectedSkill.name }}</h3>
        <p class="text-gray">作者: {{ selectedSkill.author.nickname }} | 状态: {{ getStatusText(selectedSkill.status) }}</p>
        <p>{{ selectedSkill.description }}</p>
        <div v-if="selectedSkill.skill_md" class="skill-md">
          <h4>SKILL.md</h4>
          <pre>{{ selectedSkill.skill_md }}</pre>
        </div>
      </div>
      <div v-if="selectedMCPServer" v-loading="detailLoading">
        <h3>{{ selectedMCPServer.name }}</h3>
        <p class="text-gray">作者: {{ selectedMCPServer.author.nickname }} | 状态: {{ getStatusText(selectedMCPServer.status) }}</p>
        <p>{{ selectedMCPServer.description }}</p>
        <p><a :href="selectedMCPServer.repo_url" target="_blank" rel="noopener noreferrer">查看 Git 仓库</a></p>
        <div v-if="selectedMCPServer.installations.length" class="tools-json">
          <h4>安装配置</h4>
          <pre>{{ JSON.stringify(selectedMCPServer.installations, null, 2) }}</pre>
        </div>
        <div v-if="selectedMCPServer.readme" class="readme">
          <h4>补充说明</h4>
          <pre>{{ selectedMCPServer.readme }}</pre>
        </div>
      </div>
    </el-drawer>
    <el-dialog v-model="rejectDialogVisible" title="拒绝原因" width="500px">
      <el-input v-model="rejectReason" type="textarea" :rows="4" placeholder="请输入拒绝原因（1-500字符）" />
      <template #footer>
        <el-button @click="rejectDialogVisible = false">取消</el-button>
        <el-button type="danger" @click="confirmReject">确认拒绝</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { getAdminSkills, getAdminSkill, getAdminMCPServers, getAdminMCPServer, reviewAdminSkill, reviewAdminMCPServer, type AdminResource, type AdminSkillDetail, type AdminMCPServerDetail } from '@/api/admin'

const activeTab = ref('skill')

const skills = ref<AdminResource[]>([])
const skillKeyword = ref('')
const skillStatus = ref('pending_review')
const skillPage = ref(1)
const skillPageSize = ref(20)
const skillTotal = ref(0)
const skillLoading = ref(false)

const mcpServers = ref<AdminResource[]>([])
const mcpKeyword = ref('')
const mcpStatus = ref('pending_review')
const mcpPage = ref(1)
const mcpPageSize = ref(20)
const mcpTotal = ref(0)
const mcpLoading = ref(false)

const drawerVisible = ref(false)
const drawerTitle = ref('')
const detailLoading = ref(false)
const selectedSkill = ref<AdminSkillDetail | null>(null)
const selectedMCPServer = ref<AdminMCPServerDetail | null>(null)

const rejectDialogVisible = ref(false)
const rejectReason = ref('')
const rejectTarget = ref<{ type: 'skill' | 'mcp_server'; id: number } | null>(null)

function getStatusType(status: string) {
  switch (status) {
    case 'published': return 'success'
    case 'pending_review': return 'warning'
    case 'rejected': return 'danger'
    case 'archived': return 'info'
    default: return 'info'
  }
}

function getStatusText(status: string) {
  switch (status) {
    case 'published': return '已发布'
    case 'pending_review': return '待审核'
    case 'rejected': return '已拒绝'
    case 'archived': return '已下架'
    default: return status
  }
}

async function loadSkills() {
  skillLoading.value = true
  try {
    const res = await getAdminSkills({ keyword: skillKeyword.value, status: skillStatus.value, page: skillPage.value, page_size: skillPageSize.value })
    skills.value = res.data.data.list
    skillTotal.value = res.data.data.total
  } catch {
    ElMessage.error('加载失败')
  } finally {
    skillLoading.value = false
  }
}

function handleSkillPageChange(p: number) {
  skillPage.value = p
  loadSkills()
}

async function viewSkill(row: AdminResource) {
  drawerVisible.value = true
  drawerTitle.value = 'Skill 详情'
  detailLoading.value = true
  selectedSkill.value = null
  try {
    const res = await getAdminSkill(row.id)
    selectedSkill.value = res.data.data
  } catch {
    ElMessage.error('加载详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function loadMCPServers() {
  mcpLoading.value = true
  try {
    const res = await getAdminMCPServers({ keyword: mcpKeyword.value, status: mcpStatus.value, page: mcpPage.value, page_size: mcpPageSize.value })
    mcpServers.value = res.data.data.list
    mcpTotal.value = res.data.data.total
  } catch {
    ElMessage.error('加载失败')
  } finally {
    mcpLoading.value = false
  }
}

function handleMcpPageChange(p: number) {
  mcpPage.value = p
  loadMCPServers()
}

async function viewMCPServer(row: AdminResource) {
  drawerVisible.value = true
  drawerTitle.value = 'MCP Server 详情'
  detailLoading.value = true
  selectedMCPServer.value = null
  try {
    const res = await getAdminMCPServer(row.id)
    selectedMCPServer.value = res.data.data
  } catch {
    ElMessage.error('加载详情失败')
  } finally {
    detailLoading.value = false
  }
}

async function reviewResource(type: 'skill' | 'mcp_server', id: number, approved: boolean) {
  try {
    await ElMessageBox.confirm('确认通过审核？', '审核确认')
    if (type === 'skill') {
      await reviewAdminSkill(id, { approved })
    } else {
      await reviewAdminMCPServer(id, { approved })
    }
    ElMessage.success('审核成功')
    if (type === 'skill') {
      await loadSkills()
    } else {
      await loadMCPServers()
    }
  } catch {
    // cancelled
  }
}

function showRejectDialog(type: 'skill' | 'mcp_server', id: number) {
  rejectTarget.value = { type, id }
  rejectReason.value = ''
  rejectDialogVisible.value = true
}

async function confirmReject() {
  if (!rejectTarget.value) return
  const reason = rejectReason.value.trim()
  if (reason.length === 0) {
    ElMessage.error('请填写拒绝原因')
    return
  }
  if ([...reason].length > 500) {
    ElMessage.error('拒绝原因不能超过 500 个字符')
    return
  }
  try {
    if (rejectTarget.value.type === 'skill') {
      await reviewAdminSkill(rejectTarget.value.id, { approved: false, reason })
    } else {
      await reviewAdminMCPServer(rejectTarget.value.id, { approved: false, reason })
    }
    ElMessage.success('已拒绝')
    rejectDialogVisible.value = false
    if (rejectTarget.value.type === 'skill') {
      await loadSkills()
    } else {
      await loadMCPServers()
    }
  } catch {
    ElMessage.error('操作失败')
  }
}

function handleTabChange(tab: string) {
  if (tab === 'skill') {
    loadSkills()
  } else {
    loadMCPServers()
  }
}

onMounted(() => {
  loadSkills()
})
</script>

<style scoped>
.resources-view {
  padding: 20px;
}
.text-gray {
  color: #999;
  font-size: 12px;
}
.skill-md, .tools-json, .readme {
  margin-top: 20px;
}
.skill-md pre, .tools-json pre, .readme pre {
  background: #f5f5f5;
  padding: 10px;
  border-radius: 4px;
  overflow-x: auto;
  white-space: pre-wrap;
  word-wrap: break-word;
}
</style>
